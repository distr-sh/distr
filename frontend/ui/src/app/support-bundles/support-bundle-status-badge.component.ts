import {Component, computed, input} from '@angular/core';
import {SupportBundleStatus} from '../types/support-bundle';
import {supportBundleStatusVariant} from './support-bundle-display';

@Component({
  selector: 'app-support-bundle-status-badge',
  host: {class: 'distr-badge capitalize', '[class]': 'variantClass()'},
  template: `{{ status() }}`,
})
export class SupportBundleStatusBadgeComponent {
  public readonly status = input.required<SupportBundleStatus>();
  protected readonly variantClass = computed(() => `distr-badge-${supportBundleStatusVariant(this.status())}`);
}
