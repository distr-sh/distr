import {Component, input} from '@angular/core';

import {NgClass} from '@angular/common';
import {RouterLink} from '@angular/router';
import {SupportBundle} from '../../types/support-bundle';

@Component({
  selector: 'app-support-bundle-dashboard-card',
  templateUrl: './support-bundle-dashboard-card.component.html',
  imports: [RouterLink, NgClass],
})
export class SupportBundleDashboardCardComponent {
  public readonly customerName = input.required<string>();
  public readonly bundles = input.required<SupportBundle[]>();

  protected statusDotClass(status: SupportBundle['status']): string {
    switch (status) {
      case 'initialized':
        return 'bg-blue-500';
      case 'created':
        return 'bg-yellow-400';
      case 'resolved':
        return 'bg-green-500';
      case 'canceled':
        return 'bg-gray-400';
      default:
        return '';
    }
  }
}
