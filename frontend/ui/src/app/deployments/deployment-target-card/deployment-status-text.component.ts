import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, Directive, input} from '@angular/core';
import {DeploymentRevisionStatus, DeploymentStatusType, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {never} from '../../../util/exhaust';
import {isStale} from '../../../util/model';
import {AbstractStatusDotDirective} from '../../components/status-dot';
import {DeploymentStatusBadgeComponent} from '../deployment-status-badge.component';

function currentStatus(deployment: DeploymentWithLatestRevision): DeploymentRevisionStatus | undefined {
  return deployment.currentStatus ?? deployment.latestStatus;
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
    const s = currentStatus(this.deployment());
    if (s === undefined) {
      return 'unknown';
    } else if (s.type === 'error') {
      return 'danger';
    } else if (isStale(s)) {
      return 'warning';
    } else if (s.type === 'progressing') {
      return 'info';
    } else if (s.type === 'running') {
      return 'ok-circle';
    } else if (s.type === 'healthy') {
      return 'ok';
    } else {
      return never(s.type);
    }
  });
}

@Component({
  selector: 'app-deployment-status-text',
  imports: [DeploymentStatusDotDirective, DatePipe, DeploymentStatusBadgeComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  template: `
    <div class="flex gap-1 items-center" [title]="(status()?.createdAt | date: 'short') ?? ''">
      <div class="size-3" appDeploymentStatusDot [deployment]="deployment()"></div>
      @if (status(); as drs) {
        @if (drs.type === 'error') {
          Error
        } @else if (stale()) {
          Stale
        } @else if (drs.type === 'progressing') {
          Progressing
        } @else if (drs.type === 'running') {
          Running
        } @else {
          Healthy
        }
      } @else {
        No status
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
  protected readonly stale = computed(() => {
    const status = this.status();
    return status !== undefined && isStale(status);
  });
  protected readonly pending = computed(() => pendingUpdate(this.deployment()));
}
