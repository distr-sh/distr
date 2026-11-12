import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {DeploymentTargetsComponent} from '../../../deployments/deployment-targets.component';

@Component({
  template: `<app-deployment-targets [customerOrganizationId]="customerOrganizationId()" />`,
  imports: [DeploymentTargetsComponent],
})
export class CustomerDeploymentsTabComponent {
  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );
}
