import {Component, computed, input} from '@angular/core';
import {SupportBundleStatus} from '../types/support-bundle';
import {supportBundleProgress, supportBundleProgressVariant} from './support-bundle-display';

@Component({
  selector: 'app-support-bundle-status-badge',
  host: {class: 'distr-badge', '[class]': 'variantClass()'},
  template: `
    @switch (progress()) {
      @case ('awaiting_upload') {
        Awaiting upload
      }
      @case ('uploaded') {
        Uploaded
      }
      @case ('resolved') {
        Resolved
      }
    }
  `,
})
export class SupportBundleStatusBadgeComponent {
  public readonly status = input.required<SupportBundleStatus>();
  public readonly resourceCount = input.required<number>();
  protected readonly progress = computed(() => supportBundleProgress(this.status(), this.resourceCount()));
  protected readonly variantClass = computed(() => `distr-badge-${supportBundleProgressVariant(this.progress())}`);
}
