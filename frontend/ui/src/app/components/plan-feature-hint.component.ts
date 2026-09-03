import {Component, input} from '@angular/core';
import {WEBSITE_URL} from '../../constants';
import {SubscriptionType} from '../types/subscription';
import {PlanBadgeComponent} from './plan-badge.component';

@Component({
  selector: 'app-plan-feature-hint',
  imports: [PlanBadgeComponent],
  template: `
    <div
      role="tooltip"
      class="p-3 text-sm font-medium text-gray-900 bg-white border border-gray-200 dark:bg-gray-600 dark:text-gray-200 dark:border-gray-700 rounded-lg shadow-lg max-w-64">
      <div class="mb-1">
        This feature is available to organizations with a
        <app-plan-badge [plan]="plan()" />
        subscription.
      </div>
      Our Pro and Business plans are also available for self-hosted customers. See our
      <a
        [href]="websiteUrl + '/pricing/'"
        target="_blank"
        rel="noopener noreferrer"
        class="text-gray-600 dark:text-gray-400 hover:underline"
        >pricing page</a
      >.
    </div>
  `,
})
export class PlanFeatureHintComponent {
  public readonly plan = input.required<SubscriptionType>();

  protected readonly websiteUrl = WEBSITE_URL;
}
