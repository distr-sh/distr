import {Component, computed, inject} from '@angular/core';
import {RouterLink} from '@angular/router';
import {CustomOidcComponent} from '../../organization-settings/custom-oidc/custom-oidc.component';
import {FeatureFlagService} from '../../services/feature-flag.service';

@Component({
  selector: 'app-customer-portal-domain',
  templateUrl: './customer-portal-domain.component.html',
  imports: [CustomOidcComponent, RouterLink],
})
export class CustomerPortalDomainComponent {
  private readonly featureFlags = inject(FeatureFlagService);

  protected readonly appDomainLabel = computed(() =>
    this.featureFlags.isPartnerManagementEnabled() ? 'Vendor & Partner Portal domain' : 'Vendor Portal domain'
  );
}
