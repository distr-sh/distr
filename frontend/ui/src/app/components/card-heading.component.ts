import {Component, input} from '@angular/core';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';

@Component({
  selector: 'h2[appCardHeading]',
  templateUrl: './card-heading.component.html',
  imports: [FaIconComponent],
  host: {class: 'text-lg font-semibold text-gray-900 dark:text-white'},
})
export class CardHeadingComponent {
  public readonly icon = input<IconDefinition>();
}
