import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {CustomerControllerSettingsComponent} from '../../../customer-settings/customer-controller-settings.component';
import {CustomerDomainAndSignInComponent} from '../../../customer-settings/customer-domain-and-sign-in.component';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {FeatureFlagService} from '../../../services/feature-flag.service';
import {CardHeadingComponent} from '../../card-heading.component';
import {CustomerFeatureDisabledAlertComponent} from './customer-feature-disabled-alert.component';

@Component({
  templateUrl: './customer-settings-tab.component.html',
  imports: [
    CardHeadingComponent,
    CustomerControllerSettingsComponent,
    CustomerDomainAndSignInComponent,
    CustomerFeatureDisabledAlertComponent,
  ],
})
export class CustomerSettingsTabComponent {
  private readonly featureFlags = inject(FeatureFlagService);

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  private readonly customerOrganizations = toSignal(inject(CustomerOrganizationsService).getCustomerOrganizations());
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
}
