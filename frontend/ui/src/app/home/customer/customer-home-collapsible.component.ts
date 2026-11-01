import {
  afterNextRender,
  afterRenderEffect,
  Component,
  DestroyRef,
  ElementRef,
  inject,
  input,
  model,
  signal,
  viewChild,
} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown, faChevronUp} from '@fortawesome/free-solid-svg-icons';

/** Clips the projected content to a fixed height until the user expands it. */
@Component({
  selector: 'app-customer-home-collapsible',
  templateUrl: './customer-home-collapsible.component.html',
  imports: [FaIconComponent],
  host: {
    class: 'relative flex flex-col min-h-0',
    '[class.lg:flex-1]': 'fill() && !expanded()',
  },
})
export class CustomerHomeCollapsibleComponent {
  /** On large screens, the collapsed content fills the remaining height of its flex column instead. */
  public readonly fill = input(false);
  public readonly expanded = model(false);

  protected readonly faChevronDown = faChevronDown;
  protected readonly faChevronUp = faChevronUp;

  protected readonly overflowing = signal(false);
  protected readonly collapsing = signal(false);
  /** Set while expanded, since max-height can only animate between lengths and not to or from none. */
  protected readonly maxHeight = signal<number | undefined>(undefined);
  private collapsedHeight = 0;

  private readonly clip = viewChild.required<ElementRef<HTMLElement>>('clip');
  private readonly content = viewChild.required<ElementRef<HTMLElement>>('content');

  constructor() {
    // Collapsing does not resize the clip when the content shrank below the limit while expanded, so the
    // ResizeObserver alone would keep a stale "Show more".
    afterRenderEffect(() => {
      if (!this.expanded()) {
        this.measure();
      }
    });
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      const observer = new ResizeObserver(() => this.measure());
      observer.observe(this.clip().nativeElement);
      observer.observe(this.content().nativeElement);
      destroyRef.onDestroy(() => observer.disconnect());
    });
  }

  protected expand() {
    // The clip starts at its collapsed height, which a filling clip only has as a computed value, and grows to the
    // content height in a later frame so the transition has two lengths to animate between.
    this.collapsedHeight = this.clip().nativeElement.clientHeight;
    this.maxHeight.set(this.collapsedHeight);
    this.expanded.set(true);
    requestAnimationFrame(() =>
      requestAnimationFrame(() => this.maxHeight.set(this.content().nativeElement.offsetHeight))
    );
  }

  protected collapse() {
    // Shrinks back to the height the clip had before expanding and only then returns to the collapsed layout,
    // whose height a filling clip cannot animate to.
    if (this.clip().nativeElement.clientHeight === this.collapsedHeight) {
      this.finishCollapse();
    } else {
      this.collapsing.set(true);
      this.maxHeight.set(this.collapsedHeight);
    }
  }

  protected onTransitionEnd(event: TransitionEvent) {
    if (event.target === this.clip().nativeElement && event.propertyName === 'max-height' && this.collapsing()) {
      this.finishCollapse();
    }
  }

  private finishCollapse() {
    this.collapsing.set(false);
    this.maxHeight.set(undefined);
    this.expanded.set(false);
  }

  private measure() {
    const clip = this.clip().nativeElement;
    const content = this.content().nativeElement;
    if (!this.expanded()) {
      this.overflowing.set(content.offsetHeight > clip.clientHeight);
    } else if (!this.collapsing()) {
      this.maxHeight.set(content.offsetHeight);
    }
  }
}
