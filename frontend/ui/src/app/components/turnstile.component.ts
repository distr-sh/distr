import {
  afterNextRender,
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  ElementRef,
  inject,
  input,
  output,
  viewChild,
} from '@angular/core';
import {ColorSchemeService} from '../services/color-scheme.service';

type TurnstileTheme = 'light' | 'dark';

interface TurnstileRenderOptions {
  sitekey: string;
  action: string;
  theme: TurnstileTheme;
  appearance: 'always' | 'execute' | 'interaction-only';
  callback: (token: string) => void;
  'error-callback': () => void;
  'expired-callback': () => void;
}

interface TurnstileApi {
  render(element: HTMLElement, options: TurnstileRenderOptions): string | undefined;
  reset(widgetId: string): void;
  remove(widgetId: string): void;
}

interface TurnstileWindow extends Window {
  turnstile?: TurnstileApi;
}

// The script only scans the document for widgets to render once, on load, which a route change in a single page
// application never repeats, so the widget is rendered explicitly instead.
const scriptUrl = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';

// Identifies this integration in the Turnstile analytics of the Cloudflare account.
const action = 'turnstile-spin-v2';

let api: Promise<TurnstileApi> | undefined;

function loadTurnstile(): Promise<TurnstileApi> {
  api ??= new Promise<TurnstileApi>((resolve, reject) => {
    const script = document.createElement('script');
    script.src = scriptUrl;
    script.async = true;
    script.defer = true;
    script.addEventListener('load', () => {
      const loaded = (window as TurnstileWindow).turnstile;
      if (loaded) {
        resolve(loaded);
      } else {
        reject(new Error('the Turnstile script did not install its API'));
      }
    });
    script.addEventListener('error', () => reject(new Error('failed to load the Turnstile script')));
    document.head.appendChild(script);
  }).catch((e) => {
    // Drop the cached rejection so a later attempt (e.g. after an ad blocker is disabled or a transient CDN
    // failure clears) retries the load instead of reusing the failure for the rest of the session.
    api = undefined;
    throw e;
  });
  return api;
}

/**
 * Renders the Cloudflare Turnstile widget. The token it emits has to be sent along with the form and is verified
 * by the backend, which is the only place it means anything.
 */
@Component({
  selector: 'app-turnstile',
  changeDetection: ChangeDetectionStrategy.Eager,
  host: {class: 'flex justify-center'},
  template: `<div #widget class="cf-turnstile" data-action="turnstile-spin-v2"></div>`,
})
export class TurnstileComponent {
  public readonly siteKey = input.required<string>();

  /** The token of a solved challenge, or undefined once it expired or the challenge failed. */
  public readonly token = output<string | undefined>();

  private readonly colorScheme = inject(ColorSchemeService).colorScheme;
  private readonly theme = computed<TurnstileTheme>(() => (this.colorScheme() === 'dark' ? 'dark' : 'light'));

  private readonly widget = viewChild.required<ElementRef<HTMLElement>>('widget');
  private turnstile?: TurnstileApi;
  private widgetId?: string;
  private renderedTheme?: TurnstileTheme;

  constructor() {
    afterNextRender(() => this.render());
    // The theme is fixed when a widget is rendered, so switching the color scheme means rendering it again. That
    // discards the current token: challenges are solved per widget, and this is a new one.
    effect(() => {
      const theme = this.theme();
      if (this.widgetId !== undefined && theme !== this.renderedTheme) {
        this.remove();
        this.token.emit(undefined);
        void this.render();
      }
    });
    inject(DestroyRef).onDestroy(() => this.remove());
  }

  /** Discards the current token and shows a fresh challenge. A token can only be redeemed once. */
  public reset(): void {
    if (this.widgetId !== undefined) {
      this.turnstile?.reset(this.widgetId);
      this.token.emit(undefined);
    }
  }

  private async render(): Promise<void> {
    try {
      this.turnstile = await loadTurnstile();
    } catch (e) {
      console.error(e);
      return;
    }
    const theme = this.theme();
    this.widgetId = this.turnstile.render(this.widget().nativeElement, {
      sitekey: this.siteKey(),
      action,
      theme,
      // Only shows the widget when the challenge actually has to be solved by hand.
      appearance: 'interaction-only',
      callback: (token) => this.token.emit(token),
      'error-callback': () => this.token.emit(undefined),
      'expired-callback': () => this.token.emit(undefined),
    });
    this.renderedTheme = theme;
  }

  private remove(): void {
    if (this.widgetId !== undefined) {
      this.turnstile?.remove(this.widgetId);
      this.widgetId = undefined;
      this.renderedTheme = undefined;
    }
  }
}
