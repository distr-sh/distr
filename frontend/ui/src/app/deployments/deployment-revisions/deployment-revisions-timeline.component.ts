import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, input, output} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {DeploymentRevisionResponse} from '@distr-sh/distr-sdk';
import {of, switchMap} from 'rxjs';
import {OrganizationKindPipe} from '../../../util/organization-kind';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';

@Component({
  selector: 'app-deployment-revisions-timeline',
  templateUrl: './deployment-revisions-timeline.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [DatePipe, OrganizationKindPipe],
})
export class DeploymentRevisionsTimelineComponent {
  public readonly deploymentId = input.required<string>();
  public readonly revisionSelected = output<DeploymentRevisionResponse>();

  private readonly deploymentTargets = inject(DeploymentTargetsService);

  protected readonly revisions = toSignal(
    toObservable(this.deploymentId).pipe(
      switchMap((id) => (id ? this.deploymentTargets.getRevisions(id) : of<DeploymentRevisionResponse[]>([])))
    )
  );
}
