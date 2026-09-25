import {Component, computed, input} from '@angular/core';
import {DeploymentStatusType} from '@distr-sh/distr-sdk';

@Component({
  selector: 'app-deployment-status-badge',
  styleUrl: './deployment-status-badge.component.scss',
  host: {
    class: 'distr-status-badge shrink-0',
    '[class]': 'statusClass()',
  },
  template: `
    <ng-content>
      <span class="capitalize">{{ status() }}</span>
    </ng-content>
  `,
})
export class DeploymentStatusBadgeComponent {
  public readonly status = input.required<DeploymentStatusType | 'stale'>();

  protected readonly statusClass = computed(() => `status-${this.status()}`);
}
