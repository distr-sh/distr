import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, Directive, input} from '@angular/core';
import {DeploymentRevisionStatus, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {never} from '../../../util/exhaust';
import {isStale} from '../../../util/model';
import {AbstractStatusDotDirective} from '../../components/status-dot';

function currentStatus(deployment: DeploymentWithLatestRevision): DeploymentRevisionStatus | undefined {
  return deployment.currentStatus ?? deployment.latestStatus;
}

function pendingStatus(deployment: DeploymentWithLatestRevision): DeploymentRevisionStatus | undefined {
  const {currentDeploymentRevisionId, deploymentRevisionId, latestStatus} = deployment;
  if (!currentDeploymentRevisionId || currentDeploymentRevisionId === deploymentRevisionId) {
    return undefined;
  }
  if (latestStatus?.type === 'error' || latestStatus?.type === 'progressing') {
    return latestStatus;
  }
  return undefined;
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
  imports: [DeploymentStatusDotDirective, DatePipe],
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
        @if (pending.type === 'error') {
          <span
            class="distr-status-badge ms-2 bg-red-100 text-red-800 border-red-400 dark:bg-red-900 dark:text-red-300 dark:border-red-800"
            [title]="pending.message">
            Update failed
          </span>
        } @else {
          <span
            class="distr-status-badge ms-2 bg-blue-100 text-blue-800 border-blue-400 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800"
            [title]="pending.message">
            Update in progress
          </span>
        }
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
  protected readonly pending = computed(() => pendingStatus(this.deployment()));
}
