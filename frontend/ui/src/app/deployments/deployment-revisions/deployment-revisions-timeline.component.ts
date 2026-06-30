import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, input, output} from '@angular/core';
import {rxResource} from '@angular/core/rxjs-interop';
import {DeploymentRevisionResponse, DeploymentTarget} from '@distr-sh/distr-sdk';
import {OrganizationKindPipe} from '../../../util/organization-kind';
import {UserAvatarComponent} from '../../components/user-avatar.component';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';

@Component({
  selector: 'app-deployment-revisions-timeline',
  templateUrl: './deployment-revisions-timeline.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [DatePipe, OrganizationKindPipe, UserAvatarComponent],
})
export class DeploymentRevisionsTimelineComponent {
  public readonly deploymentId = input.required<string>();
  public readonly currentRevisionId = input<string>();
  public readonly deploymentTarget = input.required<DeploymentTarget>();
  public readonly revisionSelected = output<DeploymentRevisionResponse>();

  private readonly deploymentTargets = inject(DeploymentTargetsService);

  protected readonly revisionsResource = rxResource({
    params: () => ({deploymentId: this.deploymentId()}),
    stream: ({params}) => this.deploymentTargets.getRevisions(params.deploymentId),
  });
}
