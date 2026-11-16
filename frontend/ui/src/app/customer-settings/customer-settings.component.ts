import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {PageComponent} from '../components/page.component';
import {ContextService} from '../services/context.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {CustomerControllerSettingsComponent} from './customer-controller-settings.component';
import {CustomerDomainAndSignInComponent} from './customer-domain-and-sign-in.component';

@Component({
  selector: 'app-customer-settings',
  templateUrl: './customer-settings.component.html',
  imports: [CustomerControllerSettingsComponent, CustomerDomainAndSignInComponent, PageComponent],
})
export class CustomerSettingsComponent {
  private readonly featureFlags = inject(FeatureFlagService);

  protected readonly customerOrganization = toSignal(inject(ContextService).getCustomerOrganization());
  protected readonly oidcProvidersEnabled = computed(
    () =>
      (this.featureFlags.isCustomDomainsEnabled() || this.featureFlags.isCustomOidcProvidersEnabled()) &&
      this.customerOrganization()?.features?.includes('oidc_providers')
  );
}
