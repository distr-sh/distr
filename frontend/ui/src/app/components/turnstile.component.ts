import {
  afterNextRender,
  Component,
  computed,
  DestroyRef,
  effect,
  ElementRef,
  inject,
  input,
  output,
  signal,
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

function turnstileApi(): TurnstileApi | undefined {
  return (window as TurnstileWindow).turnstile;
}

function loadTurnstile(): Promise<TurnstileApi> {
  // A failed load is not cached: a single page application session outlives it, and the next attempt gets a fresh
  // script element instead of the rejection of the first one.
  api ??= requestTurnstile().catch((e) => {
    api = undefined;
    throw e;
  });
  return api;
}

function requestTurnstile(): Promise<TurnstileApi> {
  return new Promise<TurnstileApi>((resolve, reject) => {
    const script = document.createElement('script');
    script.src = scriptUrl;
    script.async = true;
    script.defer = true;
    const fail = (message: string) => {
      script.remove();
      reject(new Error(message));
    };
    script.addEventListener('load', () => {
      const loaded = turnstileApi();
      if (loaded) {
        resolve(loaded);
      } else {
        fail('the Turnstile script did not install its API');
      }
    });
    script.addEventListener('error', () => fail('failed to load the Turnstile script'));
    document.head.appendChild(script);
  });
}

/**
 * Renders the Cloudflare Turnstile widget. The token it emits has to be sent along with the form and is verified
 * by the backend, which is the only place it means anything.
 */
@Component({
  selector: 'app-turnstile',
  host: {class: 'flex justify-center'},
  template: `<div #widget class="cf-turnstile" data-action="turnstile-spin-v2"></div>`,
})
export class TurnstileComponent {
  public readonly siteKey = input.required<string>();

  /** The token of a solved challenge, or undefined once it expired or the challenge failed. */
  public readonly token = output<string | undefined>();

  private readonly loadFailed = signal(false);

  /** Whether the challenge is unavailable, meaning no token can be obtained until the page is loaded again. */
  public readonly unavailable = this.loadFailed.asReadonly();

  private readonly colorScheme = inject(ColorSchemeService).colorScheme;
  private readonly theme = computed<TurnstileTheme>(() => (this.colorScheme() === 'dark' ? 'dark' : 'light'));

  private readonly widget = viewChild.required<ElementRef<HTMLElement>>('widget');
  private widgetId?: string;

  constructor() {
    afterNextRender(() => this.render(this.theme()));
    effect(() => {
      const theme = this.theme();
      if (this.widgetId !== undefined) {
        this.remove();
        this.token.emit(undefined);
        void this.render(theme);
      }
    });
    inject(DestroyRef).onDestroy(() => this.remove());
  }

  /** Discards the current token and shows a fresh challenge. A token can only be redeemed once. */
  public reset(): void {
    if (this.widgetId !== undefined) {
      turnstileApi()?.reset(this.widgetId);
      this.token.emit(undefined);
    }
  }

  private async render(theme: TurnstileTheme): Promise<void> {
    let turnstile: TurnstileApi;
    try {
      turnstile = await loadTurnstile();
    } catch (e) {
      console.error(e);
      this.loadFailed.set(true);
      return;
    }
    this.widgetId = turnstile.render(this.widget().nativeElement, {
      sitekey: this.siteKey(),
      action,
      theme,
      // Only shows the widget when the challenge actually has to be solved by hand.
      appearance: 'interaction-only',
      callback: (token) => this.token.emit(token),
      'error-callback': () => this.token.emit(undefined),
      'expired-callback': () => this.token.emit(undefined),
    });
  }

  private remove(): void {
    if (this.widgetId !== undefined) {
      turnstileApi()?.remove(this.widgetId);
      this.widgetId = undefined;
    }
  }
}
