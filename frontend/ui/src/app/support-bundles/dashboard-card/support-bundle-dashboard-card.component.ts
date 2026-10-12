import {ChangeDetectionStrategy, Component, input} from '@angular/core';

import {RouterLink} from '@angular/router';
import {SupportBundle} from '../../types/support-bundle';
import {SupportBundleStatusBadgeComponent} from '../support-bundle-status-badge.component';

@Component({
  selector: 'app-support-bundle-dashboard-card',
  templateUrl: './support-bundle-dashboard-card.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [RouterLink, SupportBundleStatusBadgeComponent],
})
export class SupportBundleDashboardCardComponent {
  public readonly customerName = input.required<string>();
  public readonly bundles = input.required<SupportBundle[]>();
}
