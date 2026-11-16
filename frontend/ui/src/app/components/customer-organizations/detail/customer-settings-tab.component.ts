import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {faGear, faLink, faToggleOn} from '@fortawesome/free-solid-svg-icons';
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
  host: {class: 'flex flex-col gap-4'},
})
export class CustomerSettingsTabComponent {
  protected readonly faGear = faGear;
  protected readonly faLink = faLink;
  protected readonly faToggleOn = faToggleOn;

  private readonly auth = inject(AuthService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly isAdmin = this.auth.isSuperAdmin() || this.auth.hasRole('admin');
  private readonly isCustomerSettingsEnabled = toSignal(this.featureFlags.isCustomerSettingsEnabled$, {
    initialValue: false,
  });

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  private readonly customerOrganizations = toSignal(inject(CustomerOrganizationsService).getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.parentParams()?.['customerOrganizationId'])
  );

  protected readonly settingsEnabled = computed(
    () =>
      this.isAdmin &&
      this.isCustomerSettingsEnabled() &&
      (this.customerOrganization()?.features.includes('settings') ?? false)
  );
  protected readonly domainAndSignInEnabled = computed(
    () =>
      this.settingsEnabled() &&
      (this.featureFlags.isCustomDomainsEnabled() || this.featureFlags.isCustomOidcProvidersEnabled()) &&
      (this.customerOrganization()?.features.includes('oidc_providers') ?? false)
  );
}
