import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, Directive, input} from '@angular/core';
import {DeploymentRevisionStatus, DeploymentStatusType, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {never} from '../../../util/exhaust';
import {isStale} from '../../../util/model';
import {MultiSelectOption} from '../../components/multi-select/multi-select.component';
import {AbstractStatusDotDirective} from '../../components/status-dot';
import {DeploymentStatusBadgeComponent} from '../deployment-status-badge.component';

function currentStatus(deployment: DeploymentWithLatestRevision): DeploymentRevisionStatus | undefined {
  return deployment.currentStatus ?? deployment.latestStatus;
}

export type DeploymentDisplayStatus = DeploymentStatusType | 'stale' | 'none';

export const deploymentDisplayStatusSelectOptions: MultiSelectOption<DeploymentDisplayStatus>[] = [
  {value: 'healthy', label: 'Healthy'},
  {value: 'running', label: 'Running'},
  {value: 'progressing', label: 'Progressing'},
  {value: 'stale', label: 'Stale'},
  {value: 'error', label: 'Error'},
  {value: 'none', label: 'No status'},
];

export function deploymentDisplayStatus(deployment: DeploymentWithLatestRevision): DeploymentDisplayStatus {
  const s = currentStatus(deployment);
  if (s === undefined) {
    return 'none';
  } else if (s.type === 'error') {
    return 'error';
  } else if (isStale(s)) {
    return 'stale';
  } else {
    return s.type;
  }
}

interface PendingUpdate {
  type: Extract<DeploymentStatusType, 'error' | 'progressing'>;
  message: string;
}

function pendingUpdate(deployment: DeploymentWithLatestRevision): PendingUpdate | undefined {
  const {currentDeploymentRevisionId, deploymentRevisionId, latestStatus} = deployment;
  if (!currentDeploymentRevisionId || currentDeploymentRevisionId === deploymentRevisionId) {
    return undefined;
  }
  if (latestStatus?.type === 'error') {
    return {type: 'error', message: latestStatus.message};
  }
  return {type: 'progressing', message: latestStatus?.message ?? ''};
}

@Directive({selector: '[appDeploymentStatusDot]'})
export class DeploymentStatusDotDirective extends AbstractStatusDotDirective {
  public readonly deployment = input.required<DeploymentWithLatestRevision>();
  protected override style = computed(() => {
    const status = deploymentDisplayStatus(this.deployment());
    switch (status) {
      case 'none':
        return 'unknown';
      case 'error':
        return 'danger';
      case 'stale':
        return 'warning';
      case 'progressing':
        return 'info';
      case 'running':
        return 'ok-circle';
      case 'healthy':
        return 'ok';
      default:
        return never(status);
    }
  });
}

@Component({
  selector: 'app-deployment-status-text',
  imports: [DeploymentStatusDotDirective, DatePipe, DeploymentStatusBadgeComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  template: `
    <div class="flex items-center gap-1" [title]="(status()?.createdAt | date: 'short') ?? ''">
      <div class="size-3" appDeploymentStatusDot [deployment]="deployment()"></div>
      @switch (displayStatus()) {
        @case ('error') {
          Error
        }
        @case ('stale') {
          Stale
        }
        @case ('progressing') {
          Progressing
        }
        @case ('running') {
          Running
        }
        @case ('healthy') {
          Healthy
        }
        @default {
          No status
        }
      }
      @if (pending(); as pending) {
        <app-deployment-status-badge [status]="pending.type" class="ms-2" [title]="pending.message">
          @if (pending.type === 'error') {
            Update failed
          } @else {
            Updating
          }
        </app-deployment-status-badge>
      }
    </div>
  `,
})
export class DeploymentStatusTextComponent {
  public readonly deployment = input.required<DeploymentWithLatestRevision>();
  protected readonly status = computed(() => currentStatus(this.deployment()));
  protected readonly displayStatus = computed(() => deploymentDisplayStatus(this.deployment()));
  protected readonly pending = computed(() => pendingUpdate(this.deployment()));
}
