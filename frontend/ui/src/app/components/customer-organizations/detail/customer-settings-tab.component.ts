import {Component, computed, inject, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY, firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../../util/errors';
import {CustomerControllerSettingsComponent} from '../../../customer-settings/customer-controller-settings.component';
import {CustomerDomainAndSignInComponent} from '../../../customer-settings/customer-domain-and-sign-in.component';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {FeatureFlagService} from '../../../services/feature-flag.service';
import {ToastService} from '../../../services/toast.service';
import {AlertComponent} from '../../alert/alert.component';
import {CardHeadingComponent} from '../../card-heading.component';

@Component({
  templateUrl: './customer-settings-tab.component.html',
  imports: [
    AlertComponent,
    CardHeadingComponent,
    CustomerControllerSettingsComponent,
    CustomerDomainAndSignInComponent,
  ],
})
export class CustomerSettingsTabComponent {
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly toast = inject(ToastService);

  protected readonly canEdit = inject(AuthService).hasAnyRole('read_write', 'admin');

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  private readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.parentParams()?.['customerOrganizationId'])
  );

  protected readonly settingsEnabled = computed(
    () => this.customerOrganization()?.features.includes('settings') ?? false
  );
  protected readonly domainAndSignInEnabled = computed(
    () =>
      (this.featureFlags.isCustomDomainsEnabled() || this.featureFlags.isCustomOidcProvidersEnabled()) &&
      (this.customerOrganization()?.features.includes('oidc_providers') ?? false)
  );
  protected readonly enabling = signal(false);

  protected async enableSettings() {
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
          features: [...customer.features, 'settings'],
        })
      );
      this.toast.success('Settings have been enabled for this customer');
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
