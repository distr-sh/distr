import {Component, computed, inject, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY, firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../../util/errors';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {ToastService} from '../../../services/toast.service';
import {SupportBundleListComponent} from '../../../support-bundles/list/support-bundle-list.component';
import {AlertComponent} from '../../alert/alert.component';

@Component({
  templateUrl: './customer-support-tab.component.html',
  imports: [AlertComponent, SupportBundleListComponent],
})
export class CustomerSupportTabComponent {
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly toast = inject(ToastService);

  protected readonly canEdit = inject(AuthService).hasAnyRole('read_write', 'admin');

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );

  private readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  private readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.customerOrganizationId())
  );
  protected readonly isSupportDisabled = computed(() => {
    const customer = this.customerOrganization();
    return customer !== undefined && !customer.features.includes('support_bundles');
  });
  protected readonly enabling = signal(false);

  protected async enableSupport() {
    const customer = this.customerOrganization();
    if (!customer) {
      return;
    }
    this.enabling.set(true);
    try {
      await firstValueFrom(
        this.customerOrganizationsService.updateCustomerOrganization(customer.id, {
          name: customer.name,
          imageId: customer.imageId,
          features: [...customer.features, 'support_bundles'],
        })
      );
      this.toast.success('Support has been enabled for this customer');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.enabling.set(false);
    }
  }
}
