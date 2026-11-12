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
import {combineLatest, firstValueFrom, of, switchMap} from 'rxjs';
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
import {QuickNavigationCardComponent, QuickNavigationCardType} from '../quick-navigation-card.component';
import {VendorHomeDeploymentHealth, VendorHomeUpdates} from './vendor-home';
import {toVendorHomeDeploymentHealth, toVendorHomeUpdates, vendorHomeAttentionStatuses} from './vendor-home-mappers';

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

  private readonly isVendor = this.auth.isVendor();

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
