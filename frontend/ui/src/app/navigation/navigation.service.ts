import {computed, inject, Injectable} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {CustomerOrganizationFeature, SidebarLink} from '@distr-sh/distr-sdk';
import {map, of, switchMap} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {LicenseKeysService} from '../services/license-keys.service';
import {OrganizationService} from '../services/organization.service';
import {Feature} from '../types/organization';
import {customerNavigation} from './customer-navigation';
import {NavigationContext, NavigationDefinition, NavigationSection, resolveNavigation} from './navigation';
import {partnerNavigation} from './partner-navigation';
import {vendorNavigation} from './vendor-navigation';

/**
 * The navigation of the current portal, for the sidebar and every other menu that lists its pages. Provided on
 * the logged-in shell rather than in root, because it reads the session's claims once and a login without a
 * page reload has to start over.
 */
@Injectable()
export class NavigationService {
  private readonly auth = inject(AuthService);
  private readonly organizationService = inject(OrganizationService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly contextService = inject(ContextService);
  private readonly licenseKeysService = inject(LicenseKeysService);
  private readonly definition: NavigationDefinition = this.auth.isVendor()
    ? vendorNavigation
    : this.auth.isPartner()
      ? partnerNavigation
      : this.auth.isCustomer()
        ? customerNavigation
        : [];

  // The license keys page lists only license keys, so a customer licensed through entitlements alone would land on
  // an empty page.
  private readonly hasLicenseKeys = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(
      switchMap((enabled) =>
        enabled && this.auth.isCustomer()
          ? this.licenseKeysService.list().pipe(map((licenseKeys) => licenseKeys.length > 0))
          : of(false)
      )
    ),
    {initialValue: false}
  );

  private readonly features = toSignal(this.organizationService.get().pipe(map((org) => new Set(org.features))), {
    initialValue: new Set<Feature>(),
  });

  private readonly customerFeatures = toSignal(
    this.contextService
      .getCustomerOrganization()
      .pipe(map((customerOrg) => new Set<CustomerOrganizationFeature>(customerOrg?.features))),
    {initialValue: new Set<CustomerOrganizationFeature>()}
  );

  private readonly sidebarLinks = toSignal(this.contextService.getSidebarLinks(), {
    initialValue: [] as SidebarLink[],
  });

  private readonly context = computed<NavigationContext>(() => ({
    user: {
      isAdmin: this.auth.hasRole('admin'),
      isAdminOrSuperAdmin: this.auth.isSuperAdmin() || this.auth.hasRole('admin'),
      isReadWriteOrAdmin: this.auth.hasAnyRole('read_write', 'admin'),
      isSuperAdmin: this.auth.isSuperAdmin(),
    },
    features: this.features(),
    customer: {
      features: this.customerFeatures(),
      hasLicenseKeys: this.hasLicenseKeys(),
      sidebarLinks: this.sidebarLinks(),
    },
  }));

  public readonly sections = computed<NavigationSection[]>(() => resolveNavigation(this.definition, this.context()));
}
