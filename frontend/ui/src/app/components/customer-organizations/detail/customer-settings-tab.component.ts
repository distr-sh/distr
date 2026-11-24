import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {CustomerControllerSettingsComponent} from '../../../customer-settings/customer-controller-settings.component';
import {CustomerDomainAndSignInComponent} from '../../../customer-settings/customer-domain-and-sign-in.component';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {FeatureFlagService} from '../../../services/feature-flag.service';
import {SidebarLinksComponent} from '../../../sidebar-links/sidebar-links.component';
import {CardHeadingComponent} from '../../card-heading.component';
import {CustomerFeaturesComponent} from './customer-features.component';

@Component({
  templateUrl: './customer-settings-tab.component.html',
  imports: [
    CardHeadingComponent,
    CustomerControllerSettingsComponent,
    CustomerDomainAndSignInComponent,
    CustomerFeaturesComponent,
    SidebarLinksComponent,
  ],
})
export class CustomerSettingsTabComponent {
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly auth = inject(AuthService);
  private readonly isAdmin = this.auth.isSuperAdmin() || this.auth.hasRole('admin');

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );
  private readonly customerOrganizations = toSignal(inject(CustomerOrganizationsService).getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.customerOrganizationId())
  );

  private readonly isCustomerSettingsEnabled = toSignal(this.featureFlags.isCustomerSettingsEnabled$, {
    initialValue: false,
  });
  protected readonly shouldShowCustomerSettings = computed(() => this.isAdmin && this.isCustomerSettingsEnabled());
  protected readonly settingsEnabled = computed(
    () => this.customerOrganization()?.features.includes('settings') ?? false
  );
  protected readonly domainAndSignInEnabled = computed(
    () =>
      (this.featureFlags.isCustomDomainsEnabled() || this.featureFlags.isCustomOidcProvidersEnabled()) &&
      (this.customerOrganization()?.features.includes('oidc_providers') ?? false)
  );
}
