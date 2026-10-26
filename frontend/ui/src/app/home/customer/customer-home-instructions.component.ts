import {
  afterNextRender,
  Component,
  DestroyRef,
  ElementRef,
  inject,
  input,
  output,
  signal,
  viewChild,
} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown, faChevronUp, faXmark} from '@fortawesome/free-solid-svg-icons';
import {InnerMarkdownDirective} from '../../directives/inner-markdown.directive';

@Component({
  selector: 'app-customer-home-instructions',
  templateUrl: './customer-home-instructions.component.html',
  imports: [FaIconComponent, InnerMarkdownDirective],
})
export class CustomerHomeInstructionsComponent {
  public readonly description = input.required<string>();
  public readonly dismissible = input(true);
  public readonly dismissed = output<void>();

  protected readonly faChevronDown = faChevronDown;
  protected readonly faChevronUp = faChevronUp;
  protected readonly faXmark = faXmark;

  protected readonly expanded = signal(false);
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
      observer.observe(content);
      destroyRef.onDestroy(() => observer.disconnect());
    });
  }
}
