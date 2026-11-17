import {Component, inject, input, signal} from '@angular/core';
import {CustomerOrganization, CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../../util/errors';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {ToastService} from '../../../services/toast.service';
import {AlertComponent} from '../../alert/alert.component';
import {customerFeatures} from './customer-features.component';

/** The projected content is the description. */
@Component({
  selector: 'app-customer-feature-disabled-alert',
  templateUrl: './customer-feature-disabled-alert.component.html',
  imports: [AlertComponent],
  host: {class: 'block'},
})
export class CustomerFeatureDisabledAlertComponent {
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly toast = inject(ToastService);

  public readonly customerOrganization = input.required<CustomerOrganization>();
  public readonly feature = input.required<CustomerOrganizationFeature>();
  public readonly heading = input.required<string>();
  public readonly enableLabel = input.required<string>();
  public readonly enabledMessage = input.required<string>();

  protected readonly canEdit = inject(AuthService).hasAnyRole('read_write', 'admin');
  protected readonly enabling = signal(false);

  protected async enable() {
    const customer = this.customerOrganization();
    const feature = this.feature();
    const features = new Set([...customer.features, feature]);
    const requires = customerFeatures.find((it) => it.feature === feature)?.requires;
    if (requires) {
      features.add(requires);
    }
    this.enabling.set(true);
    try {
      await firstValueFrom(
        this.customerOrganizationsService.updateCustomerOrganization(customer.id, {
          name: customer.name,
          imageId: customer.imageId,
          features: Array.from(features),
        })
      );
      this.toast.success(this.enabledMessage());
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
