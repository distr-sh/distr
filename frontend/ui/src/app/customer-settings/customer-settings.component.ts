import {Component, computed, inject} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {map, switchMap} from 'rxjs';
import {refreshInBackground} from '../../util/refresh';
import {CustomDomainComponent} from '../components/custom-domains/custom-domain/custom-domain.component';
import {IdentityProvidersComponent} from '../components/custom-domains/identity-providers/identity-providers.component';
import {PageComponent} from '../components/page.component';
import {ContextService} from '../services/context.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {CustomerControllerSettingsComponent} from './customer-controller-settings.component';

@Component({
  selector: 'app-customer-settings',
  templateUrl: './customer-settings.component.html',
  imports: [CustomerControllerSettingsComponent, CustomDomainComponent, IdentityProvidersComponent, PageComponent],
})
export class CustomerSettingsComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly contextService = inject(ContextService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly featureFlags = inject(FeatureFlagService);

  private readonly routeCustomerOrganizationId = toSignal(
    this.route.paramMap.pipe(map((params) => params.get('customerOrganizationId') ?? undefined)),
    {initialValue: undefined}
  );
  private readonly customerOrganization = toSignal(
    toObservable(this.routeCustomerOrganizationId).pipe(
      switchMap((id) =>
        id
          ? this.customerOrganizationsService
              .getCustomerOrganizations()
              .pipe(map((organizations) => organizations.find((organization) => organization.id === id)))
          : this.contextService.getCustomerOrganization()
      )
    )
  );

  protected readonly customerOrganizationId = computed(
    () => this.routeCustomerOrganizationId() ?? this.customerOrganization()?.id
  );
  protected readonly oidcProvidersEnabled = computed(
    () =>
      (this.featureFlags.isCustomDomainsEnabled() || this.featureFlags.isCustomOidcProvidersEnabled()) &&
      this.customerOrganization()?.features?.includes('oidc_providers')
  );

  constructor() {
    if (this.route.snapshot.paramMap.has('customerOrganizationId')) {
      refreshInBackground(this.customerOrganizationsService.refresh());
    }
  }
}
