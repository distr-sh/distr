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
import {combineLatest, firstValueFrom, map, of} from 'rxjs';
import {refreshInBackground} from '../../../util/refresh';
import {CardHeadingComponent} from '../../components/card-heading.component';
import {PageComponent} from '../../components/page.component';
import {ApplicationsService} from '../../services/applications.service';
import {ArtifactPullsService} from '../../services/artifact-pulls.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {QuickNavigationCardComponent} from '../quick-navigation-card.component';
import {toVendorHomeDeploymentHealth, vendorHomeAttentionStatuses} from './vendor-home-mappers';

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

  private readonly isVendor = this.auth.isVendor();

  protected readonly faAddressBook = faAddressBook;
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faCircleArrowUp = faCircleArrowUp;
  protected readonly faClockRotateLeft = faClockRotateLeft;
  protected readonly faKey = faKey;
  protected readonly faLifeRing = faLifeRing;
  protected readonly faShieldHalved = faShieldHalved;

  private readonly customerCount = toSignal(
    this.customerOrganizationsService.getCustomerOrganizations().pipe(map((customers) => customers.length))
  );
  private readonly deploymentTargets = toSignal(this.deploymentTargetsService.list());
  protected readonly deploymentHealth = computed(() => {
    const targets = this.deploymentTargets();
    return targets && toVendorHomeDeploymentHealth(targets);
  });
  private readonly deploymentCount = computed(() => this.deploymentHealth()?.total);
  protected readonly deploymentAttentionQueryParams = {status: vendorHomeAttentionStatuses};
  // Partners have no access to the applications.
  private readonly applicationCount = toSignal(
    this.isVendor ? this.applicationsService.list().pipe(map((applications) => applications.length)) : of(null)
  );

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
      ...(this.isVendor ? [this.applicationsService.refresh()] : [])
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
