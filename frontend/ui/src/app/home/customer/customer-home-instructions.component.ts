import {Component, input, output} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faXmark} from '@fortawesome/free-solid-svg-icons';
import {InnerMarkdownDirective} from '../../directives/inner-markdown.directive';
import {CustomerHomeCollapsibleComponent} from './customer-home-collapsible.component';

@Component({
  selector: 'app-customer-home-instructions',
  templateUrl: './customer-home-instructions.component.html',
  imports: [FaIconComponent, InnerMarkdownDirective, CustomerHomeCollapsibleComponent],
})
export class CustomerHomeInstructionsComponent {
  public readonly description = input.required<string>();
  public readonly dismissible = input(true);
  public readonly dismissed = output<void>();

  protected readonly faXmark = faXmark;
}
