import {afterRenderEffect, Component, ElementRef, input, output, signal, viewChild} from '@angular/core';
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

  private readonly content = viewChild.required<ElementRef<HTMLElement>>('content');

  constructor() {
    afterRenderEffect(() => {
      this.description();
      const element = this.content().nativeElement;
      if (!this.expanded()) {
        this.overflowing.set(element.scrollHeight > element.clientHeight);
      }
    });
  }
}
