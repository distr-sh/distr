import {ChangeDetectionStrategy, Component, input} from '@angular/core';

import {NgClass} from '@angular/common';
import {RouterLink} from '@angular/router';
import {SupportBundle} from '../../types/support-bundle';
import {supportBundleStatusBadgeClass} from '../support-bundle-display';

@Component({
  selector: 'app-support-bundle-dashboard-card',
  templateUrl: './support-bundle-dashboard-card.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [RouterLink, NgClass],
})
export class SupportBundleDashboardCardComponent {
  public readonly customerName = input.required<string>();
  public readonly bundles = input.required<SupportBundle[]>();

  protected readonly statusBadgeClass = supportBundleStatusBadgeClass;
}
