import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, Directive, input} from '@angular/core';
import {DeploymentRevisionStatus, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {never} from '../../../util/exhaust';
import {isStale} from '../../../util/model';
import {AbstractStatusDotDirective} from '../../components/status-dot';

function currentStatus(deployment: DeploymentWithLatestRevision): DeploymentRevisionStatus | undefined {
  return deployment.currentStatus ?? deployment.latestStatus;
}

function newestStatus(deployment: DeploymentWithLatestRevision): DeploymentRevisionStatus | undefined {
  const {currentStatus, latestStatus} = deployment;
  if (currentStatus && latestStatus) {
    return dayjs(currentStatus.createdAt).isAfter(latestStatus.createdAt) ? currentStatus : latestStatus;
  }
  return currentStatus ?? latestStatus;
}

// An agent that retries a revision it could not apply keeps reporting, so the current revision's
// status must not decay into "Stale" just because the reports are about the newer revision.
function isDeploymentStale(deployment: DeploymentWithLatestRevision): boolean {
  const status = newestStatus(deployment);
  return status !== undefined && isStale(status);
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
    const deployment = this.deployment();
    const s = currentStatus(deployment);
    if (s === undefined) {
      return 'unknown';
    } else if (s.type === 'error') {
      return 'danger';
    } else if (isDeploymentStale(deployment)) {
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
          <span class="text-xs text-red-500 dark:text-red-400" [title]="pending.message">Update failed</span>
        } @else {
          <span class="text-xs text-gray-500 dark:text-gray-400" [title]="pending.message">Update in progress</span>
        }
      }
    </div>
  `,
})
export class DeploymentStatusTextComponent {
  public readonly deployment = input.required<DeploymentWithLatestRevision>();
  protected readonly status = computed(() => currentStatus(this.deployment()));
  protected readonly stale = computed(() => isDeploymentStale(this.deployment()));
  protected readonly pending = computed(() => pendingStatus(this.deployment()));
}
