import {NgPlural, NgPluralCase} from '@angular/common';
import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, Router} from '@angular/router';
import {
  faAddressBook,
  faBoxesStacked,
  faCircleArrowUp,
  faClockRotateLeft,
  faKey,
  faLifeRing,
  faShieldHalved,
} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, firstValueFrom, map, of, switchMap} from 'rxjs';
import {refreshInBackground} from '../../../util/refresh';
import {CardHeadingComponent} from '../../components/card-heading.component';
import {PageComponent} from '../../components/page.component';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ApplicationsService} from '../../services/applications.service';
import {ArtifactPullsService} from '../../services/artifact-pulls.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {LicensesService} from '../../services/licenses.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {QuickNavigationCardComponent, QuickNavigationCardType} from '../quick-navigation-card.component';
import {VendorHomeDeploymentHealth, VendorHomeLicenses, VendorHomeUpdates} from './vendor-home';
import {
  countOpenSupportBundles,
  toVendorHomeDeploymentHealth,
  toVendorHomeLicenses,
  toVendorHomeUpdates,
  vendorHomeAttentionStatuses,
  vendorHomeLicenseExpiryWindowDays,
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

@Component({
  selector: 'app-vendor-home-page',
  templateUrl: './vendor-home-page.component.html',
  imports: [NgPlural, NgPluralCase, PageComponent, CardHeadingComponent, QuickNavigationCardComponent],
})
export class VendorHomePageComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly auth = inject(AuthService);
  private readonly artifactPullsService = inject(ArtifactPullsService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly licensesService = inject(LicensesService);
  private readonly supportBundlesService = inject(SupportBundlesService);

  protected readonly isVendor = this.auth.isVendor();

  protected readonly faAddressBook = faAddressBook;
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faCircleArrowUp = faCircleArrowUp;
  protected readonly faClockRotateLeft = faClockRotateLeft;
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
  protected readonly openSupportBundles = toSignal(
    this.featureFlags.isSupportBundlesEnabled$.pipe(
      switchMap((enabled) =>
        enabled ? this.supportBundlesService.list().pipe(map((bundles) => countOpenSupportBundles(bundles))) : of(null)
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

  protected readonly counts = computed(() => {
    const customers = this.customerCount();
    const deployments = this.deploymentCount();
    const applications = this.applicationCount();
    return customers === undefined || deployments === undefined || applications === undefined
      ? undefined
      : {customers, deployments, applications};
  });

  constructor() {
    refreshInBackground(
      this.customerOrganizationsService.refresh(),
      this.deploymentTargetsService.refresh(),
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
