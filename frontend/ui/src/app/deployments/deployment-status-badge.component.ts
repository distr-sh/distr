import {Component, computed, input} from '@angular/core';
import {DeploymentStatusType} from '@distr-sh/distr-sdk';
import {never} from '../../util/exhaust';
import {BadgeVariant} from '../components/badge';

@Component({
  selector: 'app-deployment-status-badge',
  host: {
    class: 'distr-badge shrink-0',
    '[class]': 'variantClass()',
  },
  template: `
    <ng-content>
      <span class="capitalize">{{ status() }}</span>
    </ng-content>
  `,
})
export class DeploymentStatusBadgeComponent {
  public readonly status = input.required<DeploymentStatusType | 'stale'>();

  protected readonly variantClass = computed(() => `distr-badge-${deploymentStatusVariant(this.status())}`);
}

function deploymentStatusVariant(status: DeploymentStatusType | 'stale'): BadgeVariant {
  switch (status) {
    case 'healthy':
    case 'running':
      return 'success';
    case 'progressing':
      return 'info';
    case 'error':
      return 'error';
    case 'stale':
      return 'warning';
    default:
      return never(status);
  }
}
