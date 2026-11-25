import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  signal,
  TemplateRef,
  viewChild,
} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup} from '@angular/forms';
import {ActivatedRoute, Params, Router} from '@angular/router';
import {Application, ApplicationVersion, DeploymentTarget, DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPlus} from '@fortawesome/free-solid-svg-icons';
import {
  catchError,
  combineLatest,
  combineLatestWith,
  first,
  firstValueFrom,
  map,
  Observable,
  of,
  shareReplay,
  startWith,
  switchMap,
} from 'rxjs';
import {arraysEqual, compareBy} from '../../util/arrays';
import {syncSearchWithQueryParam} from '../../util/filter';
import {refreshInBackground} from '../../util/refresh';
import {isDeploymentOutdated} from '../../util/versions';
import {MultiSelectComponent, MultiSelectOption} from '../components/multi-select/multi-select.component';
import {SearchBarComponent} from '../components/search-bar.component';
import {ApplicationEntitlementsService} from '../services/application-entitlements.service';
import {ApplicationsService} from '../services/applications.service';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {DeploymentTargetsMetricsService} from '../services/deployment-target-metrics.service';
import {DeploymentTargetsService} from '../services/deployment-targets.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {OrganizationService} from '../services/organization.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ApplicationEntitlement} from '../types/application-entitlement';
import {customerFeatureRequiredMessage} from '../types/customer-features';
import {DeploymentTargetLatestMetrics} from '../types/deployment-target-metrics';
import {UNLIMITED_QTY} from '../types/subscription';
import {DeploymentModalComponent} from './deployment-modal.component';
import {
  deploymentDisplayStatus,
  DeploymentStatusFilter,
  deploymentStatusFilterSelectOptions,
  isDeploymentStatusFilter,
} from './deployment-target-card/deployment-status-text.component';
import {DeploymentTargetCardComponent} from './deployment-target-card/deployment-target-card.component';
import {deploymentTargetSearchTexts, matchesSearch} from './deployment-target-search';
import {DeploymentWizardComponent} from './deployment-wizard/deployment-wizard.component';

export interface DeploymentTargetViewData extends DeploymentTarget {
  metrics?: DeploymentTargetLatestMetrics;
}

// customer organization ids are UUIDs, so this cannot collide with one
const internalCustomerOption = 'internal';
const deploymentStatusFilterLabels = new Map(
  deploymentStatusFilterSelectOptions.map((option) => [option.value, option.label])
);

function deploymentStatusFilters(
  deployment: DeploymentWithLatestRevision,
  applications: Application[],
  entitlements: ApplicationEntitlement[]
): DeploymentStatusFilter[] {
  const status = deploymentDisplayStatus(deployment);
  return isDeploymentOutdated(deployment, applications, entitlements) ? [status, 'outdated'] : [status];
}

@Component({
  selector: 'app-deployment-targets',
  imports: [
    AsyncPipe,
    FaIconComponent,
    DeploymentWizardComponent,
    OverlayModule,
    DeploymentTargetCardComponent,
    DeploymentModalComponent,
    SearchBarComponent,
    MultiSelectComponent,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './deployment-targets.component.html',
  host: {'[hidden]': 'isHidden()'},
})
export class DeploymentTargetsComponent implements AfterViewInit {
  public readonly auth = inject(AuthService);
  private readonly overlay = inject(OverlayService);
  private readonly applications = inject(ApplicationsService);
  private readonly applicationEntitlements = inject(ApplicationEntitlementsService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly deploymentTargets = inject(DeploymentTargetsService);
  private readonly deploymentTargetMetrics = inject(DeploymentTargetsMetricsService);
  private readonly organizationService = inject(OrganizationService);
  private readonly customerOrganizations = inject(CustomerOrganizationsService);
  private readonly context = inject(ContextService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  protected readonly plusIcon = faPlus;

  private modal?: DialogRef;

  protected readonly deploymentWizard = viewChild.required<TemplateRef<unknown>>('deploymentWizard');
  protected readonly deploymentModal = viewChild.required<TemplateRef<unknown>>('deploymentModal');

  selectedDeploymentTarget = signal<DeploymentTarget | undefined>(undefined);
  selectedDeployment = signal<DeploymentWithLatestRevision | undefined>(undefined);
  selectedApplicationVersionId = signal<string | undefined>(undefined);

  readonly filterForm = new FormGroup({
    search: new FormControl(this.route.snapshot.queryParamMap.get('search') ?? ''),
  });

  public readonly customerOrganizationId = input<string>();
  /** Blocks creating deployments for a customer that lacks the feature, and hides the component unless it has some. */
  public readonly customerFeatureDisabled = input(false);

  private readonly isVendorOrPartner = this.auth.isVendor() || this.auth.isPartner();
  protected readonly canFilterByCustomer = computed(() => this.isVendorOrPartner && !this.customerOrganizationId());
  private readonly queryParams = toSignal(this.route.queryParamMap, {requireSync: true});
  // Equal by content, so that changing one filter does not restart the poll that depends on another.
  protected readonly selectedCustomerIds = computed(
    () => {
      const customerOrganizationId = this.customerOrganizationId();
      const params = this.queryParams();
      if (customerOrganizationId) {
        return [customerOrganizationId];
      } else if (this.canFilterByCustomer()) {
        return [
          ...(params.get('internal') === 'true' ? [internalCustomerOption] : []),
          ...params.getAll('customerOrganizationId'),
        ];
      } else {
        return [];
      }
    },
    {equal: arraysEqual}
  );
  protected readonly selectedApplicationIds = computed(() => this.queryParams().getAll('applicationId'), {
    equal: arraysEqual,
  });
  protected readonly selectedStatuses = computed(
    () => this.queryParams().getAll('status').filter(isDeploymentStatusFilter),
    {equal: arraysEqual}
  );
  protected readonly statusSelectOptions = deploymentStatusFilterSelectOptions;

  protected readonly customerSelectOptions = toSignal(
    this.isVendorOrPartner
      ? this.customerOrganizations.getCustomerOrganizations().pipe(
          map((customers) => [
            // Only vendors have deployment targets that belong to no customer.
            ...(this.auth.isVendor() ? [{value: internalCustomerOption, label: 'Internal'}] : []),
            ...[...customers]
              .sort(compareBy((co) => co.name))
              .map<MultiSelectOption>((co) => ({value: co.id, label: co.name})),
          ])
        )
      : of([]),
    {initialValue: []}
  );

  private readonly applications$ = this.applications.list();
  private readonly applicationEntitlements$ = this.featureFlags.isLicensingEnabled$.pipe(
    switchMap((enabled) => (enabled ? this.applicationEntitlements.list() : of([])))
  );

  protected readonly applicationSelectOptions = toSignal(
    this.applications$.pipe(
      map((apps) =>
        apps.flatMap<MultiSelectOption>((app) => (app.id ? [{value: app.id, label: app.name ?? app.id}] : []))
      )
    ),
    {initialValue: []}
  );

  private readonly deploymentFilter = computed(() => ({
    applicationIds: this.selectedApplicationIds(),
    statuses: this.selectedStatuses(),
  }));

  readonly deploymentTargets$ = toObservable(this.selectedCustomerIds).pipe(
    switchMap((ids) =>
      this.deploymentTargets.poll({
        customerOrganizationIds: ids.filter((id) => id !== internalCustomerOption),
        internal: ids.includes(internalCustomerOption),
      })
    ),
    shareReplay({bufferSize: 1, refCount: true}),
    takeUntilDestroyed()
  );
  readonly deploymentTargetMetrics$ = this.deploymentTargetMetrics.poll().pipe(
    takeUntilDestroyed(),
    catchError(() => of([]))
  );

  private readonly organization = toSignal(this.organizationService.get());
  protected readonly limit = computed(() => {
    const org = this.organization();
    return !(org && org.subscriptionLimits) ? undefined : org.subscriptionLimits.maxDeploymentsPerCustomerOrganization;
  });
  protected readonly toolbarLimit = computed(() => {
    const customerIds = this.selectedCustomerIds();
    if (this.customerOrganizationId()) {
      return this.limit();
    } else if (this.auth.isPartner() || (customerIds.length > 0 && !customerIds.includes(internalCustomerOption))) {
      return undefined;
    } else {
      return this.limit();
    }
  });
  protected readonly count = toSignal(
    combineLatest([this.deploymentTargets$, this.context.getUser()]).pipe(
      map(([targets, user]) => {
        const customerOrganizationId = this.customerOrganizationId() ?? user.customerOrganizationId;
        return targets.filter((it) => it.customerOrganization?.id === customerOrganizationId).length;
      })
    ),
    {initialValue: 0}
  );
  private readonly deploymentTargetCount = toSignal(this.deploymentTargets$.pipe(map((targets) => targets.length)));
  protected readonly isHidden = computed(() => this.customerFeatureDisabled() && !this.deploymentTargetCount());
  protected readonly createBlockedReason = computed(() =>
    this.customerFeatureDisabled() ? customerFeatureRequiredMessage(['deployment_targets']) : undefined
  );
  protected readonly limitReached = computed(() => {
    const limit = this.toolbarLimit();
    return limit !== undefined && limit !== UNLIMITED_QTY && this.count() >= limit;
  });

  protected readonly filteredDeploymentTargets$: Observable<DeploymentTargetViewData[]> = combineLatest([
    this.deploymentTargets$,
    this.applications$.pipe(startWith([])),
    this.applicationEntitlements$.pipe(startWith([])),
    this.filterForm.controls.search.valueChanges.pipe(startWith(this.filterForm.controls.search.value)),
    toObservable(this.deploymentFilter),
  ]).pipe(
    map(([deploymentTargets, applications, entitlements, search, {applicationIds, statuses}]) => {
      const query = search?.toLowerCase();
      const needsStatuses = !!query || statuses.length > 0;
      return deploymentTargets.filter((dt) => {
        const deployments = dt.deployments.map((d) => ({
          deployment: d,
          statuses: needsStatuses ? deploymentStatusFilters(d, applications, entitlements) : [],
        }));
        const isSearchMatch =
          !query ||
          matchesSearch(
            [
              ...deploymentTargetSearchTexts(dt, this.organization()?.name),
              ...deployments.flatMap(({statuses}) =>
                statuses.map((status) => deploymentStatusFilterLabels.get(status))
              ),
            ],
            query
          );
        const matchesFilter =
          (applicationIds.length === 0 && statuses.length === 0) ||
          deployments.some(
            ({deployment, statuses: deploymentStatuses}) =>
              (applicationIds.length === 0 ||
                (deployment.application.id !== undefined && applicationIds.includes(deployment.application.id))) &&
              (statuses.length === 0 || deploymentStatuses.some((status) => statuses.includes(status)))
          );
        return isSearchMatch && matchesFilter;
      });
    }),
    combineLatestWith(this.deploymentTargetMetrics$),
    map(([deploymentTargets, deploymentTargetMetrics]) =>
      deploymentTargets.map((dt) => ({
        ...dt,
        metrics: deploymentTargetMetrics.find((x) => x.deploymentTargetId === dt.id),
      }))
    ),
    map((deploymentTargets) => deploymentTargets.sort(compareBy((dt) => dt.customerOrganization?.name ?? '')))
  );

  constructor() {
    refreshInBackground(
      this.applications.refresh(),
      firstValueFrom(this.featureFlags.isLicensingEnabled$).then((enabled) =>
        enabled ? this.applicationEntitlements.refresh() : undefined
      )
    );
    syncSearchWithQueryParam(this.filterForm.controls.search);
  }

  protected selectCustomers(ids: string[]) {
    const customerOrganizationIds = ids.filter((id) => id !== internalCustomerOption);
    this.setQueryParams({
      customerOrganizationId: customerOrganizationIds.length > 0 ? customerOrganizationIds : null,
      internal: ids.includes(internalCustomerOption) || null,
    });
  }

  protected selectApplications(ids: string[]) {
    this.setQueryParams({applicationId: ids.length > 0 ? ids : null});
  }

  protected selectStatuses(statuses: DeploymentStatusFilter[]) {
    this.setQueryParams({status: statuses.length > 0 ? statuses : null});
  }

  private setQueryParams(queryParams: Params) {
    this.router.navigate([], {relativeTo: this.route, queryParams, queryParamsHandling: 'merge', replaceUrl: true});
  }

  ngAfterViewInit() {
    if (this.auth.isCustomer() && this.auth.hasAnyRole('read_write', 'admin')) {
      combineLatest([this.applications$, this.deploymentTargets$])
        .pipe(first())
        .subscribe(([apps, dts]) => {
          if (apps.length > 0 && dts.length === 0) {
            this.openWizard();
          }
        });
    }
  }

  protected showDeploymentModal(
    deploymentTarget: DeploymentTarget,
    deployment: DeploymentWithLatestRevision,
    version: ApplicationVersion
  ) {
    this.selectedDeploymentTarget.set(deploymentTarget);
    this.selectedDeployment.set(deployment);
    this.selectedApplicationVersionId.set(version?.id);
    this.hideModal();
    this.modal = this.overlay.showModal(this.deploymentModal(), {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  protected openWizard() {
    this.hideModal();
    this.modal = this.overlay.showModal(this.deploymentWizard(), {
      hasBackdrop: true,
      backdropStyleOnly: true,
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  protected hideModal(): void {
    this.modal?.close();
  }
}
