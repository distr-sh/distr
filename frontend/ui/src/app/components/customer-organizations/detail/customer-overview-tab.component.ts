import {NgPlural, NgPluralCase} from '@angular/common';
import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {faBoxesStacked, faCircleArrowUp, faKey, faLifeRing, faSliders} from '@fortawesome/free-solid-svg-icons';
import {EMPTY, firstValueFrom, map, of, switchMap} from 'rxjs';
import {refreshInBackground} from '../../../../util/refresh';
import {DeploymentStatusFilter} from '../../../deployments/deployment-target-card/deployment-status-text.component';
import {QuickNavigationCardComponent, QuickNavigationCardType} from '../../../home/quick-navigation-card.component';
import {DeploymentHealth} from '../../../home/vendor/vendor-home';
import {
  filterOpenSupportBundles,
  licenseExpiryWindowDays,
  toDeploymentHealth,
  vendorHomeAttentionStatuses,
} from '../../../home/vendor/vendor-home-mappers';
import {ActivityFeedService} from '../../../services/activity-feed.service';
import {ApplicationEntitlementsService} from '../../../services/application-entitlements.service';
import {ApplicationsService} from '../../../services/applications.service';
import {ArtifactPullsService} from '../../../services/artifact-pulls.service';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../../services/deployment-targets.service';
import {FeatureFlagService} from '../../../services/feature-flag.service';
import {LicensesService} from '../../../services/licenses.service';
import {SupportBundlesService} from '../../../services/support-bundles.service';
import {toActivityFeedItems} from '../../activity-feed/activity-feed';
import {ActivityFeedComponent} from '../../activity-feed/activity-feed.component';
import {CardHeadingComponent} from '../../card-heading.component';
import {LoadingCardComponent} from '../../loading-card/loading-card.component';
import {CustomerDetailTabId} from './customer-detail-navigation';
import {CustomerFeaturesComponent} from './customer-features.component';
import {
  CustomerOverviewLicense,
  CustomerOverviewUpdates,
  toAffectedTargetNames,
  toCustomerOverviewLicense,
  toCustomerOverviewUpdates,
} from './customer-overview-mappers';

const maxAffectedNames = 2;

function deploymentsCardType(health: DeploymentHealth): QuickNavigationCardType {
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

function updatesCardType(updates: CustomerOverviewUpdates): QuickNavigationCardType {
  if (updates.outdated > 0) {
    return 'warning';
  } else if (updates.source !== 'none') {
    return 'success';
  } else {
    return 'neutral';
  }
}

function licenseCardType(license: CustomerOverviewLicense): QuickNavigationCardType {
  switch (license.status) {
    case 'expired':
      return 'error';
    case 'expiring':
      return 'warning';
    case 'no_license':
      return license.isLicensingInUse ? 'warning' : 'neutral';
    case 'valid':
      return 'success';
  }
}

function licenseCardCount(license: CustomerOverviewLicense, type: QuickNavigationCardType): number | undefined {
  if (license.status === 'expired') {
    return license.expired;
  } else if (license.status === 'expiring') {
    return license.expiring;
  } else if (type === 'neutral') {
    return undefined;
  } else {
    return license.grants;
  }
}

@Component({
  templateUrl: './customer-overview-tab.component.html',
  imports: [
    NgPlural,
    NgPluralCase,
    CardHeadingComponent,
    CustomerFeaturesComponent,
    ActivityFeedComponent,
    LoadingCardComponent,
    QuickNavigationCardComponent,
  ],
})
export class CustomerOverviewTabComponent {
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly artifactPullsService = inject(ArtifactPullsService);
  private readonly licensesService = inject(LicensesService);
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly activityFeedService = inject(ActivityFeedService);

  protected readonly isVendor = inject(AuthService).isVendor();
  protected readonly activityViewer = this.isVendor ? 'vendor' : 'partner';
  protected readonly licenseExpiryWindowDays = licenseExpiryWindowDays;
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faCircleArrowUp = faCircleArrowUp;
  protected readonly faKey = faKey;
  protected readonly faLifeRing = faLifeRing;
  protected readonly faSliders = faSliders;
  protected readonly deploymentAttentionQueryParams = {status: vendorHomeAttentionStatuses};
  protected readonly outdatedDeploymentsQueryParams = {status: ['outdated'] satisfies DeploymentStatusFilter[]};

  private readonly customerOrganizationId$ = (inject(ActivatedRoute).parent?.params ?? EMPTY).pipe(
    map((params) => params['customerOrganizationId'] as string)
  );
  private readonly customerOrganizationId = toSignal(this.customerOrganizationId$);
  private readonly customerOrganizations = toSignal(inject(CustomerOrganizationsService).getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.customerOrganizationId())
  );
  protected readonly tabLinks = computed(() => {
    const id = this.customerOrganizationId();
    const link = (tab: CustomerDetailTabId) => ['/customers', id, tab];
    return {
      deployments: link('deployments'),
      registry: link('registry'),
      licenses: link('licenses'),
      support: link('support'),
    };
  });

  private readonly allDeploymentTargets = toSignal(this.deploymentTargetsService.list());
  private readonly deploymentTargets = computed(() => {
    const id = this.customerOrganizationId();
    return this.allDeploymentTargets()?.filter((target) => target.customerOrganization?.id === id);
  });
  private readonly applications = toSignal(this.applicationsService.list());
  private readonly applicationEntitlements = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(
      switchMap((enabled) => (enabled ? this.applicationEntitlementsService.list() : of([])))
    )
  );
  private readonly allLatestArtifactPulls = toSignal(this.artifactPullsService.listLatest());
  private readonly latestArtifactPulls = computed(() => {
    const id = this.customerOrganizationId();
    return this.allLatestArtifactPulls()?.filter((pull) => pull.customerOrganizationId === id);
  });

  protected readonly deployments = computed(() => {
    const customer = this.customerOrganization();
    const targets = this.deploymentTargets();
    if (customer === undefined || targets === undefined) {
      return undefined;
    }
    const health = toDeploymentHealth(targets);
    const type = deploymentsCardType(health);
    const affected = toAffectedTargetNames(targets, type === 'error' ? 'error' : 'stale');
    return {
      ...health,
      type,
      isDisabled: !customer.features.includes('deployment_targets'),
      affectedNames: affected.slice(0, maxAffectedNames),
      moreAffected: Math.max(affected.length - maxAffectedNames, 0),
    };
  });

  protected readonly updates = computed(() => {
    const targets = this.deploymentTargets();
    const applications = this.applications();
    const entitlements = this.applicationEntitlements();
    const pulls = this.latestArtifactPulls();
    if (targets === undefined || applications === undefined || entitlements === undefined || pulls === undefined) {
      return undefined;
    }
    const updates = toCustomerOverviewUpdates(targets, applications, entitlements, pulls);
    return {...updates, type: updatesCardType(updates)};
  });

  /** `null` while the organization has no licensing. */
  private readonly licenses = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(
      switchMap((enabled) => (enabled ? this.licensesService.list() : of(null)))
    )
  );
  protected readonly licenseCard = computed(() => {
    const licenses = this.licenses();
    const id = this.customerOrganizationId();
    if (licenses === undefined || licenses === null) {
      return licenses;
    } else if (id === undefined) {
      return undefined;
    }
    const license = toCustomerOverviewLicense(id, licenses);
    const type = licenseCardType(license);
    return {...license, type, count: licenseCardCount(license, type)};
  });

  /** `null` while the organization's plan has no support bundles. */
  private readonly supportBundles = toSignal(
    this.featureFlags.isSupportBundlesEnabled$.pipe(
      switchMap((enabled) =>
        enabled ? this.customerOrganizationId$.pipe(switchMap((id) => this.supportBundlesService.list(id))) : of(null)
      )
    )
  );
  protected readonly supportCard = computed(() => {
    const bundles = this.supportBundles();
    const customer = this.customerOrganization();
    if (bundles === undefined || bundles === null) {
      return bundles;
    } else if (customer === undefined) {
      return undefined;
    }
    const open = filterOpenSupportBundles(bundles).length;
    const isDisabled = !customer.features.includes('support_bundles');
    const type: QuickNavigationCardType =
      open > 0 ? 'warning' : !isDisabled && bundles.length > 0 ? 'success' : 'neutral';
    return {open, total: bundles.length, isDisabled, type};
  });

  protected readonly activity = toSignal(
    this.customerOrganizationId$.pipe(
      switchMap((id) => this.activityFeedService.listForCustomer(id)),
      map(toActivityFeedItems)
    )
  );

  constructor() {
    refreshInBackground(
      this.deploymentTargetsService.refresh(),
      this.applicationsService.refresh(),
      this.artifactPullsService.refreshLatest(),
      firstValueFrom(this.featureFlags.isLicensingEnabled$).then((enabled) =>
        enabled ? this.applicationEntitlementsService.refresh() : undefined
      )
    );
  }
}
