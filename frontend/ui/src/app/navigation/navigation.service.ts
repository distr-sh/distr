import {computed, inject, Injectable, signal} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {CustomerOrganizationFeature, SidebarLink} from '@distr-sh/distr-sdk';
import {map, of, switchMap} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {LicenseKeysService} from '../services/license-keys.service';
import {OrganizationService} from '../services/organization.service';
import {TutorialsService} from '../services/tutorials.service';
import {customerNavigation} from './customer-navigation';
import {
  NavigationContext,
  NavigationDefinition,
  NavigationFeature,
  NavigationSection,
  resolveNavigation,
} from './navigation';
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
  private readonly tutorialsService = inject(TutorialsService);

  private readonly definition: NavigationDefinition = this.auth.isVendor()
    ? vendorNavigation
    : this.auth.isPartner()
      ? partnerNavigation
      : this.auth.isCustomer()
        ? customerNavigation
        : [];

  private readonly isAllTutorialsStarted =
    this.auth.isVendor() && this.auth.hasAnyRole('admin')
      ? toSignal(
          toObservable(this.organizationService.hasSubscription).pipe(
            switchMap((hasSub) => (hasSub ? of(true) : this.tutorialsService.allStarted$))
          ),
          {initialValue: true}
        )
      : signal(true);

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

  private readonly isLicensingEnabled = toSignal(this.featureFlags.isLicensingEnabled$, {initialValue: false});
  private readonly isNotificationsEnabled = toSignal(this.featureFlags.isNotificationsEnabled$, {
    initialValue: false,
  });
  private readonly isSupportBundlesEnabled = toSignal(this.featureFlags.isSupportBundlesEnabled$, {
    initialValue: false,
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

  private readonly features = computed(() =>
    enabled<NavigationFeature>({
      licensing: this.isLicensingEnabled(),
      notifications: this.isNotificationsEnabled(),
      support_bundles: this.isSupportBundlesEnabled(),
      vulnerabilities: this.featureFlags.isVulnerabilitiesEnabled(),
      vendor_billing: this.featureFlags.isVendorBillingEnabled(),
      partner_management: this.featureFlags.isPartnerManagementEnabled(),
    })
  );

  private readonly context = computed<NavigationContext>(() => ({
    user: {
      isAdmin: this.auth.hasRole('admin'),
      isAdminOrSuperAdmin: this.auth.isSuperAdmin() || this.auth.hasRole('admin'),
      isReadWriteOrAdmin: this.auth.hasAnyRole('read_write', 'admin'),
      isSuperAdmin: this.auth.isSuperAdmin(),
    },
    organization: {
      hasSubscription: this.organizationService.hasSubscription(),
      hasUnstartedTutorials: !this.isAllTutorialsStarted(),
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

function enabled<T extends string>(flags: Record<T, boolean>): ReadonlySet<T> {
  return new Set((Object.keys(flags) as T[]).filter((flag) => flags[flag]));
}
