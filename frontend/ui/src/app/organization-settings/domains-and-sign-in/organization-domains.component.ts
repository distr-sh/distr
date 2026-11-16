import {Component, computed, inject} from '@angular/core';
import {RouterLink} from '@angular/router';
import {CustomDomainComponent} from '../../components/custom-domains/custom-domain/custom-domain.component';
import {IdentityProvidersComponent} from '../../components/custom-domains/identity-providers/identity-providers.component';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {CustomEmailComponent} from './custom-email.component';

@Component({
  selector: 'app-organization-domains',
  templateUrl: './organization-domains.component.html',
  imports: [CustomDomainComponent, IdentityProvidersComponent, CustomEmailComponent, RouterLink],
})
export class OrganizationDomainsComponent {
  private readonly featureFlags = inject(FeatureFlagService);

  protected readonly domainsEnabled = computed(
    () => this.featureFlags.isCustomDomainsEnabled() || this.featureFlags.isCustomOidcProvidersEnabled()
  );
  protected readonly customEmailsEnabled = computed(() => this.featureFlags.isCustomEmailsEnabled());
  protected readonly partnerManagementEnabled = computed(() => this.featureFlags.isPartnerManagementEnabled());
  protected readonly portalName = computed(() =>
    this.partnerManagementEnabled() ? 'Vendor & Partner Portal' : 'Vendor Portal'
  );
}
