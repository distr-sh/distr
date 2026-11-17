import {Component, computed, input} from '@angular/core';
import {customerLicenseStatusVariant} from './customer-list-display';
import {CustomerLicenseStatus} from './customer-list-mappers';

@Component({
  selector: 'app-customer-license-badge',
  host: {class: 'distr-badge', '[class]': 'variantClass()'},
  template: `
    @switch (status()) {
      @case ('no_license') {
        No license
      }
      @case ('expired') {
        Expired
      }
      @case ('expiring') {
        Expiring
      }
      @case ('valid') {
        Valid
      }
    }
  `,
})
export class CustomerLicenseBadgeComponent {
  public readonly status = input.required<CustomerLicenseStatus>();
  protected readonly variantClass = computed(() => `distr-badge-${customerLicenseStatusVariant(this.status())}`);
}
