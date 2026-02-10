import {Component, input} from '@angular/core';
import {DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {IsStalePipe} from '../../../util/model';
import {DeploymentStatusDotDirective} from '../../components/status-dot';

@Component({
  selector: 'app-deployment-status-text',
  imports: [DeploymentStatusDotDirective, IsStalePipe],
  template: `
    <div class="flex items-center">
      <div class="size-4 me-1" appDeploymentStatusDot [deployment]="deployment()"></div>
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
