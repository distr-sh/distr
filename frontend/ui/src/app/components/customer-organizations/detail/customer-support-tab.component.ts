import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {SupportBundleListComponent} from '../../../support-bundles/list/support-bundle-list.component';
import {CustomerFeatureDisabledAlertComponent} from './customer-feature-disabled-alert.component';

@Component({
  templateUrl: './customer-support-tab.component.html',
  imports: [CustomerFeatureDisabledAlertComponent, SupportBundleListComponent],
})
export class CustomerSupportTabComponent {
  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );

  private readonly customerOrganizations = toSignal(inject(CustomerOrganizationsService).getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.customerOrganizationId())
  );
  protected readonly isSupportDisabled = computed(() => {
    const customer = this.customerOrganization();
    return customer !== undefined && !customer.features.includes('support_bundles');
  });
}
