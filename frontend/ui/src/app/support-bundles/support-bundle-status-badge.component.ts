import {Component, computed, input} from '@angular/core';
import {SupportBundleStatus} from '../types/support-bundle';
import {
  supportBundleProgress,
  supportBundleProgressLabel,
  supportBundleProgressVariant,
} from './support-bundle-display';

@Component({
  selector: 'app-support-bundle-status-badge',
  host: {class: 'distr-badge', '[class]': 'variantClass()'},
  template: `{{ label() }}`,
})
export class SupportBundleStatusBadgeComponent {
  public readonly status = input.required<SupportBundleStatus>();
  public readonly resourceCount = input.required<number>();
  protected readonly progress = computed(() => supportBundleProgress(this.status(), this.resourceCount()));
  protected readonly label = computed(() => supportBundleProgressLabel(this.progress()));
  protected readonly variantClass = computed(() => `distr-badge-${supportBundleProgressVariant(this.progress())}`);
}
