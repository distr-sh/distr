import {NgTemplateOutlet} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, computed, inject, linkedSignal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {faClockRotateLeft} from '@fortawesome/free-solid-svg-icons';
import {
  catchError,
  combineLatest,
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
import {getFormDisplayedError} from '../../util/errors';
import {LoadingCardComponent} from '../components/loading-card/loading-card.component';
import {PageComponent} from '../components/page.component';
import {ApplicationsService} from '../services/applications.service';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {OrganizationBrandingService} from '../services/organization-branding.service';
import {OrganizationService} from '../services/organization.service';
import {OverviewService} from '../services/overview.service';
import {ToastService} from '../services/toast.service';
import {OverviewAccessTokensComponent} from './overview-access-tokens.component';
import {OverviewActivityComponent} from './overview-activity.component';
import {OverviewAlertsComponent} from './overview-alerts.component';
import {OverviewChecklistComponent} from './overview-checklist.component';
import {OverviewDeploymentsComponent, OverviewPrimarySection} from './overview-deployments.component';
import {OverviewInstructionsComponent} from './overview-instructions.component';
import {OverviewLicenseComponent} from './overview-license.component';
import {OverviewSupportBundlesComponent} from './overview-support-bundles.component';

function storedInstructionsHidden(key: string | undefined): boolean | undefined {
  const stored = key ? sessionStorage.getItem(key) : null;
  return stored === null ? undefined : stored === 'true';
}

@Component({
  selector: 'app-overview-page',
  templateUrl: './overview-page.component.html',
  imports: [
    NgTemplateOutlet,
    LoadingCardComponent,
    PageComponent,
    OverviewInstructionsComponent,
    OverviewDeploymentsComponent,
    OverviewChecklistComponent,
    OverviewLicenseComponent,
    OverviewAlertsComponent,
    OverviewAccessTokensComponent,
    OverviewSupportBundlesComponent,
    OverviewActivityComponent,
  ],
})
export class OverviewPageComponent {
  private readonly overview = inject(OverviewService);
  private readonly auth = inject(AuthService);
  private readonly context = inject(ContextService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly applications = inject(ApplicationsService);
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
      map((customerOrganization) => `overview.instructionsHidden.${customerOrganization?.id ?? 'none'}`)
    )
  );
  private readonly deploymentsEnabled = computed(() => this.features()?.includes('deployment_targets') ?? false);
  private readonly artifactsEnabled = computed(() => this.features()?.includes('artifacts') ?? false);

  private readonly reloadDeploymentTargets$ = new Subject<void>();
  protected readonly deploymentTargets = toSignal(
    this.whenFeature('deployment_targets', [], () =>
      this.reloadDeploymentTargets$.pipe(
        startWith(undefined),
        switchMap(() => this.overview.getDeploymentTargets().pipe(this.recover([])))
      )
    )
  );

  protected readonly artifacts = toSignal(
    this.whenFeature('artifacts', {artifacts: [], hasPulled: false}, () =>
      this.overview.getArtifacts().pipe(this.recover({artifacts: [], hasPulled: false}))
    )
  );

  private readonly hasApplications = toSignal(
    this.whenFeature('deployment_targets', false, () =>
      this.canWrite
        ? this.applications.list().pipe(
            map((applications) => applications.length > 0),
            this.recover(false)
          )
        : of(false)
    )
  );
  private readonly canDeploy = computed(
    () => this.deploymentsEnabled() && this.canWrite && this.hasApplications() === true
  );

  protected readonly primarySection = computed<OverviewPrimarySection | undefined>(() => {
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

  protected readonly checklistSection = computed<OverviewPrimarySection | undefined>(() => {
    if (this.features() === undefined || this.hasApplications() === undefined) {
      return undefined;
    }
    if (this.deploymentsEnabled()) {
      return this.canDeploy() ? 'deployments' : 'none';
    }
    return this.artifactsEnabled() ? 'artifacts' : 'none';
  });

  protected readonly checklist = toSignal(this.overview.getChecklist().pipe(this.recover(null)));
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
      switchMap((enabled) => (enabled ? this.overview.getLicense().pipe(this.recover(undefined)) : of(undefined)))
    )
  );

  protected readonly alerts = toSignal(
    combineLatest([this.features$, this.featureFlags.isNotificationsEnabled$.pipe(take(1))]).pipe(
      switchMap(([features, enabled]) =>
        enabled && features.includes('alerts') ? this.overview.getAlerts().pipe(this.recover(undefined)) : of(undefined)
      )
    )
  );

  protected readonly accessTokens = toSignal(this.overview.getAccessTokens().pipe(this.recover([])), {
    initialValue: [],
  });

  private readonly reloadSupportBundles$ = new Subject<void>();
  protected readonly supportBundles = toSignal(
    combineLatest([this.features$, this.featureFlags.isSupportBundlesEnabled$.pipe(take(1))]).pipe(
      switchMap(([features, enabled]) =>
        enabled && features.includes('support_bundles')
          ? this.reloadSupportBundles$.pipe(
              startWith(undefined),
              switchMap(() => this.overview.getSupportBundles().pipe(this.recover([])))
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

  protected readonly activity = toSignal(this.overview.getActivity().pipe(this.recover([])));
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
