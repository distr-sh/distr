import {Component, input} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircle} from '@fortawesome/free-regular-svg-icons';
import {faChevronRight, faCircleCheck} from '@fortawesome/free-solid-svg-icons';

export interface CustomerHomeChecklistItem {
  label: string;
  hint: string;
  done: boolean;
  link?: string;
}

@Component({
  selector: '[appCustomerHomeChecklistItem]',
  templateUrl: './customer-home-checklist-item.component.html',
  imports: [FaIconComponent],
  host: {class: 'flex items-start gap-3 p-2'},
})
export class CustomerHomeChecklistItemComponent {
  public readonly item = input.required<CustomerHomeChecklistItem>({alias: 'appCustomerHomeChecklistItem'});

  protected readonly faCircle = faCircle;
  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faChevronRight = faChevronRight;
}
