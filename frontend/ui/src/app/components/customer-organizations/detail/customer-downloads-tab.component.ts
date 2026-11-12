import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {ArtifactPullsComponent} from '../../../artifacts/artifact-pulls/artifact-pulls.component';

@Component({
  template: `<app-artifact-pulls [customerOrganizationId]="customerOrganizationId()" />`,
  imports: [ArtifactPullsComponent],
})
export class CustomerDownloadsTabComponent {
  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );
}
