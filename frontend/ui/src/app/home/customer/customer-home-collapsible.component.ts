import {
  afterNextRender,
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
  /** The full height of the content, which the expanded clip transitions to since max-height cannot animate to none. */
  protected readonly contentHeight = signal(0);

  private readonly clip = viewChild.required<ElementRef<HTMLElement>>('clip');
  private readonly content = viewChild.required<ElementRef<HTMLElement>>('content');

  constructor() {
    const destroyRef = inject(DestroyRef);
    afterNextRender(() => {
      const clip = this.clip().nativeElement;
      const content = this.content().nativeElement;
      const observer = new ResizeObserver(() => {
        this.contentHeight.set(content.offsetHeight);
        if (!this.expanded()) {
          this.overflowing.set(content.offsetHeight > clip.clientHeight);
        }
      });
      observer.observe(clip);
      observer.observe(content);
      destroyRef.onDestroy(() => observer.disconnect());
    });
  }
}
