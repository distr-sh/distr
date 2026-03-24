import {Component, computed, Directive, input} from '@angular/core';
import {DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {never} from '../../../util/exhaust';
import {isStale, IsStalePipe} from '../../../util/model';
import {AbstractStatusDotDirective} from '../../components/status-dot';

@Directive({selector: '[appDeploymentStatusDot]'})
export class DeploymentStatusDotDirective extends AbstractStatusDotDirective {
  public readonly deployment = input.required<DeploymentWithLatestRevision>();
  protected override style = computed(() => {
    const s = this.deployment().latestStatus;
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
  imports: [DeploymentStatusDotDirective, IsStalePipe],
  template: `
    <div class="flex gap-1 items-center">
      <div class="size-3" appDeploymentStatusDot [deployment]="deployment()"></div>
      @if (deployment().latestStatus; as drs) {
        @if (drs.type === 'error') {
          Error
        } @else if (drs | isStale) {
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
    </div>
  `,
})
export class DeploymentStatusTextComponent {
  public readonly deployment = input.required<DeploymentWithLatestRevision>();
}
