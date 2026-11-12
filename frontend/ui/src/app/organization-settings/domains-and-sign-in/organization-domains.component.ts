import {Component, computed, inject} from '@angular/core';
import {RouterLink} from '@angular/router';
import {CustomDomainComponent} from '../../components/custom-domains/custom-domain/custom-domain.component';
import {IdentityProvidersComponent} from '../../components/custom-domains/identity-providers/identity-providers.component';
import {FeatureFlagService} from '../../services/feature-flag.service';

@Component({
  selector: 'app-organization-domains',
  templateUrl: './organization-domains.component.html',
  imports: [CustomDomainComponent, IdentityProvidersComponent, RouterLink],
})
export class OrganizationDomainsComponent {
  private readonly featureFlags = inject(FeatureFlagService);

  protected readonly partnerManagementEnabled = computed(() => this.featureFlags.isPartnerManagementEnabled());
  protected readonly portalName = computed(() =>
    this.partnerManagementEnabled() ? 'Vendor & Partner Portal' : 'Vendor Portal'
  );
}
