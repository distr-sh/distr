import {Component, computed, inject} from '@angular/core';
import {CustomDomainComponent} from '../../components/custom-domains/custom-domain/custom-domain.component';
import {IdentityProvidersComponent} from '../../components/custom-domains/identity-providers/identity-providers.component';
import {FeatureFlagService} from '../../services/feature-flag.service';

@Component({
  selector: 'app-customer-portal-domain',
  templateUrl: './customer-portal-domain.component.html',
  imports: [CustomDomainComponent, IdentityProvidersComponent],
})
export class CustomerPortalDomainComponent {
  private readonly featureFlags = inject(FeatureFlagService);

  protected readonly appDomainLabel = computed(() =>
    this.featureFlags.isPartnerManagementEnabled() ? 'Vendor & Partner Portal domain' : 'Vendor Portal domain'
  );
}
