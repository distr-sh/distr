import {NgTemplateOutlet} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, computed, inject, linkedSignal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {faClockRotateLeft} from '@fortawesome/free-solid-svg-icons';
import {
  catchError,
  combineLatest,
  from,
  map,
  Observable,
  of,
  OperatorFunction,
  shareReplay,
  startWith,
  Subject,
  switchMap,
  take,
} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {LoadingCardComponent} from '../../components/loading-card/loading-card.component';
import {PageComponent} from '../../components/page.component';
import {AccessTokensService} from '../../services/access-tokens.service';
import {ActivityFeedService} from '../../services/activity-feed.service';
import {AlertConfigurationsService} from '../../services/alert-configurations.service';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ApplicationsService} from '../../services/applications.service';
import {ArtifactEntitlementsService} from '../../services/artifact-entitlements.service';
import {ArtifactsService} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {ContextService} from '../../services/context.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {LicenseKeysService} from '../../services/license-keys.service';
import {NotificationRecordsService} from '../../services/notification-records.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {OrganizationService} from '../../services/organization.service';
import {SecretsService} from '../../services/secrets.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {UsersService} from '../../services/users.service';
import {CustomerHomeAccessTokensComponent} from './customer-home-access-tokens.component';
import {CustomerHomeActivityComponent} from './customer-home-activity.component';
import {CustomerHomeAlertsComponent} from './customer-home-alerts.component';
import {CustomerHomeChecklistComponent} from './customer-home-checklist.component';
import {CustomerHomeDeploymentsComponent, CustomerHomePrimarySection} from './customer-home-deployments.component';
import {CustomerHomeInstructionsComponent} from './customer-home-instructions.component';
import {CustomerHomeLicenseComponent} from './customer-home-license.component';
import {
  customerHomeNotificationsSince,
  toCustomerHomeAccessTokens,
  toCustomerHomeActivity,
  toCustomerHomeAlerts,
  toCustomerHomeArtifacts,
  toCustomerHomeChecklist,
  toCustomerHomeDeploymentTargets,
  toCustomerHomeLicense,
  toCustomerHomeSupportBundles,
} from './customer-home-mappers';
import {CustomerHomeSupportBundlesComponent} from './customer-home-support-bundles.component';

function storedInstructionsHidden(key: string | undefined): boolean | undefined {
  const stored = key ? sessionStorage.getItem(key) : null;
  return stored === null ? undefined : stored === 'true';
}

@Component({
  selector: 'app-customer-home-page',
  templateUrl: './customer-home-page.component.html',
  imports: [
    NgTemplateOutlet,
    LoadingCardComponent,
    PageComponent,
    CustomerHomeInstructionsComponent,
    CustomerHomeDeploymentsComponent,
    CustomerHomeChecklistComponent,
    CustomerHomeLicenseComponent,
    CustomerHomeAlertsComponent,
    CustomerHomeAccessTokensComponent,
    CustomerHomeSupportBundlesComponent,
    CustomerHomeActivityComponent,
  ],
})
export class CustomerHomePageComponent {
  private readonly auth = inject(AuthService);
  private readonly context = inject(ContextService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly artifactEntitlementsService = inject(ArtifactEntitlementsService);
  private readonly artifactsService = inject(ArtifactsService);
  private readonly usersService = inject(UsersService);
  private readonly secretsService = inject(SecretsService);
  private readonly accessTokensService = inject(AccessTokensService);
  private readonly licenseKeysService = inject(LicenseKeysService);
  private readonly alertConfigurationsService = inject(AlertConfigurationsService);
  private readonly notificationRecordsService = inject(NotificationRecordsService);
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly activityFeedService = inject(ActivityFeedService);
  private readonly organizationBranding = inject(OrganizationBrandingService);
  private readonly organizationService = inject(OrganizationService);
  private readonly toast = inject(ToastService);

  protected readonly faClockRotateLeft = faClockRotateLeft;

  protected readonly canWrite = this.auth.hasAnyRole('read_write', 'admin');
  protected readonly isAdmin = this.auth.hasRole('admin');

  private readonly customerOrganization$ = this.context.getCustomerOrganization().pipe(take(1), shareReplay(1));
  private readonly features$ = this.customerOrganization$.pipe(
    map((customerOrganization) => customerOrganization?.features ?? [])
  );
  private readonly features = toSignal(this.features$);
  // Session storage outlives a logout, so the key is per customer to keep one account's choice from leaking
  // into the next one signed in in the same tab.
  private readonly instructionsHiddenKey = toSignal(
    this.customerOrganization$.pipe(
      map((customerOrganization) => `home.instructionsHidden.${customerOrganization?.id ?? 'none'}`)
    )
  );
  private readonly deploymentsEnabled = computed(() => this.features()?.includes('deployment_targets') ?? false);
  private readonly artifactsEnabled = computed(() => this.features()?.includes('artifacts') ?? false);

  private readonly applicationEntitlements$ = this.featureFlags.isLicensingEnabled$.pipe(
    take(1),
    switchMap((enabled) => (enabled ? this.applicationEntitlementsService.list() : of([]))),
    shareReplay(1)
  );

  private readonly reloadDeploymentTargets$ = new Subject<void>();
  private readonly rawDeploymentTargets$ = this.whenFeature('deployment_targets', [], () =>
    this.reloadDeploymentTargets$.pipe(
      startWith(undefined),
      switchMap(() => this.deploymentTargetsService.refresh())
    )
  ).pipe(shareReplay({bufferSize: 1, refCount: true}));

  protected readonly deploymentTargets = toSignal(
    this.whenFeature('deployment_targets', [], () =>
      combineLatest([this.rawDeploymentTargets$, this.applicationsService.list(), this.applicationEntitlements$]).pipe(
        map(([targets, applications, entitlements]) =>
          toCustomerHomeDeploymentTargets(targets, applications, entitlements)
        ),
        this.recover([])
      )
    )
  );

  private readonly rawArtifacts$ = this.whenFeature('artifacts', [], () => from(this.artifactsService.refresh())).pipe(
    shareReplay(1)
  );

  protected readonly artifacts = toSignal(
    this.rawArtifacts$.pipe(map(toCustomerHomeArtifacts), this.recover({artifacts: [], hasPulled: false}))
  );

  private readonly hasApplications = toSignal(
    this.whenFeature('deployment_targets', false, () =>
      this.canWrite
        ? this.applicationsService.list().pipe(
            map((applications) => applications.length > 0),
            this.recover(false)
          )
        : of(false)
    )
  );
  private readonly canDeploy = computed(
    () => this.deploymentsEnabled() && this.canWrite && this.hasApplications() === true
  );

  protected readonly primarySection = computed<CustomerHomePrimarySection | undefined>(() => {
    const targets = this.deploymentTargets();
    const artifacts = this.artifacts();
    if (!targets || !artifacts || this.hasApplications() === undefined) {
      return undefined;
    }
    if (targets.length > 0) {
      return 'deployments';
    }
    if (artifacts.hasPulled && artifacts.artifacts.length > 0) {
      return 'artifacts';
    }
    if (this.canDeploy()) {
      return 'deployments';
    }
    if (this.artifactsEnabled() && !artifacts.hasPulled && artifacts.artifacts.length > 0) {
      return 'artifacts';
    }
    return 'none';
  });

  private readonly branding = toSignal(
    this.organizationBranding.get().pipe(
      catchError((e) => {
        if (!(e instanceof HttpErrorResponse && e.status === 404)) {
          this.showError(e);
        }
        return of(undefined);
      })
    )
  );
  protected readonly vendorLogo = computed(() => this.branding()?.logoImageId);
  protected readonly instructions = computed(() => this.branding()?.description || undefined);
  private readonly instructionsHiddenChoice = linkedSignal(() =>
    storedInstructionsHidden(this.instructionsHiddenKey())
  );
  private readonly instructionsHiddenByDefault = computed(
    () => !!(this.deploymentTargets()?.some((target) => target.deployments.length > 0) || this.artifacts()?.hasPulled)
  );
  private readonly instructionsHidden = computed(
    () => this.instructionsHiddenChoice() ?? this.instructionsHiddenByDefault()
  );
  // Without a primary section the instructions are all the page has to show.
  protected readonly instructionsDismissible = computed(() => this.primarySection() !== 'none');
  protected readonly visibleInstructions = computed(() =>
    this.primarySection() === undefined || (this.instructionsHidden() && this.instructionsDismissible())
      ? undefined
      : this.instructions()
  );
  protected readonly instructionsRestorable = computed(
    () => this.instructions() !== undefined && this.instructionsHidden() && this.instructionsDismissible()
  );

  protected readonly checklistSection = computed<CustomerHomePrimarySection | undefined>(() => {
    if (this.features() === undefined || this.hasApplications() === undefined) {
      return undefined;
    }
    if (this.deploymentsEnabled()) {
      return this.canDeploy() ? 'deployments' : 'none';
    }
    return this.artifactsEnabled() ? 'artifacts' : 'none';
  });

  private readonly accessTokens$ = this.accessTokensService.list().pipe(shareReplay(1));

  protected readonly checklist = toSignal(
    combineLatest({
      user: this.context.getUser(),
      userCount: this.usersService.getUsers().pipe(map((users) => users.length)),
      deploymentTargets: this.rawDeploymentTargets$,
      secretCount: this.whenFeature('deployment_targets', 0, () =>
        this.secretsService.list().pipe(map((secrets) => secrets.length))
      ),
      accessTokenCount: this.accessTokens$.pipe(map((tokens) => tokens.length)),
      artifacts: this.rawArtifacts$,
    }).pipe(map(toCustomerHomeChecklist), this.recover(null))
  );
  protected readonly checklistCompleted = computed(() => {
    const checklist = this.checklist();
    const section = this.checklistSection();
    return (
      !!checklist &&
      checklist.profileCompleted &&
      checklist.teamMembersInvited &&
      (section !== 'deployments' || (checklist.deploymentCreated && checklist.secretCreated)) &&
      (section !== 'artifacts' || (checklist.accessTokenCreated && checklist.artifactPulled))
    );
  });

  protected readonly license = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(
      take(1),
      switchMap((enabled) =>
        enabled
          ? combineLatest([
              this.licenseKeysService.list(),
              this.applicationEntitlements$,
              this.artifactEntitlementsService.list(),
            ]).pipe(
              map(([licenseKeys, applicationEntitlements, artifactEntitlements]) =>
                toCustomerHomeLicense(licenseKeys, [...applicationEntitlements, ...artifactEntitlements])
              ),
              this.recover(undefined)
            )
          : of(undefined)
      )
    )
  );

  protected readonly alerts = toSignal(
    combineLatest([this.features$, this.featureFlags.isNotificationsEnabled$.pipe(take(1))]).pipe(
      switchMap(([features, enabled]) =>
        enabled && features.includes('alerts')
          ? combineLatest([
              this.alertConfigurationsService.list(),
              this.notificationRecordsService.list(undefined, customerHomeNotificationsSince()),
            ]).pipe(
              map(([configurations, records]) => toCustomerHomeAlerts(configurations, records)),
              this.recover(undefined)
            )
          : of(undefined)
      )
    )
  );

  protected readonly accessTokens = toSignal(
    this.accessTokens$.pipe(
      map((tokens) => toCustomerHomeAccessTokens(tokens)),
      this.recover([])
    ),
    {initialValue: []}
  );

  private readonly reloadSupportBundles$ = new Subject<void>();
  protected readonly supportBundles = toSignal(
    combineLatest([this.features$, this.featureFlags.isSupportBundlesEnabled$.pipe(take(1))]).pipe(
      switchMap(([features, enabled]) =>
        enabled && features.includes('support_bundles')
          ? this.reloadSupportBundles$.pipe(
              startWith(undefined),
              switchMap(() =>
                this.supportBundlesService.list().pipe(map(toCustomerHomeSupportBundles), this.recover([]))
              )
            )
          : of(undefined)
      )
    )
  );

  // The checklist is requested for every customer, so until it has loaded the panels next to the activity
  // are assumed to exist, which keeps the activity from moving out of the full width once they appear.
  protected readonly hasPanels = computed(
    () =>
      this.checklist() === undefined ||
      (!!this.checklist() && this.checklistSection() !== undefined && !this.checklistCompleted()) ||
      this.license() !== undefined ||
      this.alerts() !== undefined ||
      this.accessTokens().length > 0 ||
      this.supportBundles() !== undefined
  );

  protected readonly activity = toSignal(
    this.activityFeedService.list().pipe(map(toCustomerHomeActivity), this.recover([]))
  );
  protected readonly activityShown = computed(() => this.activity()?.length !== 0);
  protected readonly vendorName = toSignal(this.organizationService.get().pipe(map((org) => org.name)));

  protected setInstructionsHidden(hidden: boolean) {
    const key = this.instructionsHiddenKey();
    if (key) {
      sessionStorage.setItem(key, String(hidden));
    }
    this.instructionsHiddenChoice.set(hidden);
  }

  protected reloadDeploymentTargets() {
    this.reloadDeploymentTargets$.next();
  }

  protected reloadSupportBundles() {
    this.reloadSupportBundles$.next();
  }

  private whenFeature<T>(
    feature: CustomerOrganizationFeature,
    disabled: T,
    enabled: () => Observable<T>
  ): Observable<T> {
    return this.features$.pipe(switchMap((features) => (features.includes(feature) ? enabled() : of(disabled))));
  }

  private recover<T, F>(fallback: F): OperatorFunction<T, T | F> {
    return catchError((e) => {
      this.showError(e);
      return of(fallback);
    });
  }

  private showError(e: unknown) {
    const msg = getFormDisplayedError(e);
    if (msg) {
      this.toast.error(msg);
    }
  }
}
