import {NgPlural, NgPluralCase} from '@angular/common';
import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, Router} from '@angular/router';
import {
  faAddressBook,
  faBoxesStacked,
  faCircleArrowUp,
  faKey,
  faLifeRing,
  faShieldHalved,
} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, firstValueFrom, map, of, switchMap} from 'rxjs';
import {recoverWithToast} from '../../../util/errors';
import {refreshInBackground} from '../../../util/refresh';
import {CardHeadingComponent} from '../../components/card-heading.component';
import {LoadingCardComponent} from '../../components/loading-card/loading-card.component';
import {PageComponent} from '../../components/page.component';
import {ActivityFeedService} from '../../services/activity-feed.service';
import {AdvisoriesService} from '../../services/advisories.service';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ApplicationsService} from '../../services/applications.service';
import {ArtifactPullsService} from '../../services/artifact-pulls.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {LicensesService} from '../../services/licenses.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {toHomeActivity} from '../home-activity';
import {HomeActivityComponent} from '../home-activity.component';
import {QuickNavigationCardComponent, QuickNavigationCardType} from '../quick-navigation-card.component';
import {VendorHomeDeploymentHealth, VendorHomeLicenses, VendorHomeUpdates} from './vendor-home';
import {
  countOpenSupportBundles,
  toVendorHomeAdvisoryCounts,
  toVendorHomeDeploymentHealth,
  toVendorHomeLicenses,
  toVendorHomeUpdates,
  vendorHomeAttentionStatuses,
  vendorHomeLicenseExpiryWindowDays,
  vendorHomeOpenAdvisoryStatuses,
} from './vendor-home-mappers';

function deploymentHealthType(health: VendorHomeDeploymentHealth): QuickNavigationCardType {
  if (health.error > 0) {
    return 'error';
  } else if (health.stale > 0) {
    return 'warning';
  } else if (health.healthy + health.running > 0) {
    return 'success';
  } else {
    return 'neutral';
  }
}

function updatesCardType(updates: VendorHomeUpdates): QuickNavigationCardType {
  if (updates.outdatedCustomers > 0) {
    return 'warning';
  } else if (updates.totalCustomers > 0) {
    return 'success';
  } else {
    return 'neutral';
  }
}

function licensesCardType(licenses: VendorHomeLicenses): QuickNavigationCardType {
  if (licenses.expiredCustomerIds.length > 0) {
    return 'error';
  } else if (licenses.expiringCustomerIds.length > 0) {
    return 'warning';
  } else {
    return 'success';
  }
}

function supportCardType(open: number, isConfigured: boolean): QuickNavigationCardType {
  if (open > 0) {
    return 'warning';
  } else if (isConfigured) {
    return 'success';
  } else {
    return 'neutral';
  }
}

@Component({
  selector: 'app-vendor-home-page',
  templateUrl: './vendor-home-page.component.html',
  imports: [
    NgPlural,
    NgPluralCase,
    PageComponent,
    CardHeadingComponent,
    LoadingCardComponent,
    QuickNavigationCardComponent,
    HomeActivityComponent,
  ],
})
export class VendorHomePageComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly auth = inject(AuthService);
  private readonly artifactPullsService = inject(ArtifactPullsService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly activityFeedService = inject(ActivityFeedService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly toast = inject(ToastService);

  private readonly licensesService = inject(LicensesService);

  private readonly supportBundlesService = inject(SupportBundlesService);

  private readonly advisoriesService = inject(AdvisoriesService);

  protected readonly isVendor = this.auth.isVendor();
  protected readonly activityViewer = this.isVendor ? 'vendor' : 'partner';
  protected readonly canConfigureSupport = this.isVendor && this.auth.hasAnyRole('read_write', 'admin');
  protected readonly faAddressBook = faAddressBook;
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faCircleArrowUp = faCircleArrowUp;
  protected readonly faKey = faKey;
  protected readonly faLifeRing = faLifeRing;
  protected readonly faShieldHalved = faShieldHalved;

  private readonly customers = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  private readonly customerCount = computed(() => this.customers()?.length);
  private readonly deploymentTargets = toSignal(this.deploymentTargetsService.list());
  protected readonly deploymentHealth = computed(() => {
    const targets = this.deploymentTargets();
    if (targets === undefined) {
      return undefined;
    }
    const health = toVendorHomeDeploymentHealth(targets);
    return {...health, type: deploymentHealthType(health)};
  });
  private readonly deploymentCount = computed(() => this.deploymentHealth()?.total);
  protected readonly deploymentAttentionQueryParams = {status: vendorHomeAttentionStatuses};
  private readonly applications = toSignal(this.applicationsService.list());
  private readonly applicationCount = computed(() => {
    const applications = this.applications();
    // Partners have no applications page, so their summary leaves the applications out.
    return applications === undefined ? undefined : this.isVendor ? applications.length : null;
  });
  private readonly applicationEntitlements = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(
      switchMap((enabled) => (enabled ? this.applicationEntitlementsService.list() : of([])))
    )
  );

  protected readonly licenseExpiryWindowDays = vendorHomeLicenseExpiryWindowDays;
  /** `null` while the organization has no licensing. */
  private readonly licenseCustomers = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(
      switchMap((enabled) =>
        enabled ? this.licensesService.list().pipe(map((licenses) => toVendorHomeLicenses(licenses))) : of(null)
      )
    )
  );
  protected readonly licenseCard = computed(() => {
    const licenses = this.licenseCustomers();
    if (licenses === undefined || licenses === null) {
      return licenses;
    }
    const ids = [...licenses.expiredCustomerIds, ...licenses.expiringCustomerIds];
    return {
      customers: ids.length,
      expired: licenses.expiredCustomerIds.length,
      expiring: licenses.expiringCustomerIds.length,
      type: licensesCardType(licenses),
      link: ids.length === 1 ? ['/licenses', ids[0]] : ids.length > 1 ? ['/customers'] : ['/licenses'],
    };
  });

  /** `null` while the organization's plan has no support bundles. */
  private readonly supportState = toSignal(
    this.featureFlags.isSupportBundlesEnabled$.pipe(
      switchMap((enabled) =>
        enabled
          ? combineLatest([
              this.supportBundlesService.list(),
              // Partners cannot read the configuration, and the support page shows them no banner about it either.
              this.isVendor ? this.supportBundlesService.hasConfiguration() : of(true),
            ]).pipe(map(([bundles, isConfigured]) => ({open: countOpenSupportBundles(bundles), isConfigured})))
          : of(null)
      )
    )
  );
  private readonly latestArtifactPulls = toSignal(this.artifactPullsService.getLatest());

  protected readonly updates = computed(() => {
    const customers = this.customers();
    const targets = this.deploymentTargets();
    const applications = this.applications();
    const entitlements = this.applicationEntitlements();
    const pulls = this.latestArtifactPulls();
    if (
      customers === undefined ||
      targets === undefined ||
      applications === undefined ||
      entitlements === undefined ||
      pulls === undefined
    ) {
      return undefined;
    }
    const updates = toVendorHomeUpdates(customers, targets, applications, entitlements, pulls);
    return {...updates, type: updatesCardType(updates)};
  });

  protected readonly supportCard = computed(() => {
    const state = this.supportState();
    if (state === undefined || state === null) {
      return state;
    }
    const {open, isConfigured} = state;
    const shouldConfigure = open === 0 && !isConfigured && this.canConfigureSupport;
    return {
      open,
      type: supportCardType(open, isConfigured),
      link: shouldConfigure ? '/support/settings' : '/support',
    };
  });

  /** `null` while the organization's plan has no vulnerabilities. */
  protected readonly advisoryCounts = toSignal(
    this.isVendor
      ? this.featureFlags.isVulnerabilitiesEnabled$.pipe(
          switchMap((enabled) =>
            enabled ? this.advisoriesService.list().pipe(map(toVendorHomeAdvisoryCounts)) : of(null)
          )
        )
      : of(null)
  );

  protected readonly openAdvisoryQueryParams = {status: vendorHomeOpenAdvisoryStatuses};

  protected readonly counts = computed(() => {
    const customers = this.customerCount();
    const deployments = this.deploymentCount();
    const applications = this.applicationCount();
    return customers === undefined || deployments === undefined || applications === undefined
      ? undefined
      : {customers, deployments, applications};
  });

  protected readonly activity = toSignal(
    this.activityFeedService.list().pipe(map(toHomeActivity), recoverWithToast(this.toast, []))
  );

  constructor() {
    refreshInBackground(
      this.customerOrganizationsService.refresh(),
      this.deploymentTargetsService.refresh(),
      this.activityFeedService.refresh(),
      this.applicationsService.refresh(),
      firstValueFrom(this.featureFlags.isLicensingEnabled$).then((enabled) =>
        enabled ? this.applicationEntitlementsService.refresh() : undefined
      )
    );
    if (this.route.snapshot.queryParamMap.get('from') === 'login') {
      this.redirectAfterLogin();
    }
  }

  private async redirectAfterLogin() {
    if (this.isVendor) {
      const [pulls, deploymentTargets] = await firstValueFrom(
        combineLatest([this.artifactPullsService.get({count: 1}), this.deploymentTargetsService.list()])
      );
      if (pulls.length === 0 && deploymentTargets.length === 0) {
        await this.router.navigate(['/tutorials']);
        return;
      }
    }
    await this.router.navigate([], {relativeTo: this.route, queryParams: {from: null}, replaceUrl: true});
  }
}
