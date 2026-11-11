import {Component, inject} from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {customerDetailTabs} from './customer-detail-navigation';

@Component({
  template: `<p class="distr-text-muted text-sm">{{ label }}</p>`,
})
export class CustomerTabPlaceholderComponent {
  private readonly path = inject(ActivatedRoute).snapshot.routeConfig?.path;
  protected readonly label = customerDetailTabs.find((tab) => tab.id === this.path)?.label;
}
