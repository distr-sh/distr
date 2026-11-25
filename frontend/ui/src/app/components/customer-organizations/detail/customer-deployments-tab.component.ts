import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {DeploymentTargetsComponent} from '../../../deployments/deployment-targets.component';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {CustomerFeatureDisabledAlertComponent} from './customer-feature-disabled-alert.component';

@Component({
  templateUrl: './customer-deployments-tab.component.html',
  imports: [CustomerFeatureDisabledAlertComponent, DeploymentTargetsComponent],
})
export class CustomerDeploymentsTabComponent {
  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );

  private readonly customerOrganizations = toSignal(inject(CustomerOrganizationsService).getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.customerOrganizationId())
  );
  protected readonly isDeploymentsDisabled = computed(() => {
    const customer = this.customerOrganization();
    return customer !== undefined && !customer.features.includes('deployment_targets');
  });
}
