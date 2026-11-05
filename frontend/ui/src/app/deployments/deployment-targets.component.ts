import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  signal,
  TemplateRef,
  viewChild,
} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup} from '@angular/forms';
import {ActivatedRoute, Params, Router} from '@angular/router';
import {
  ApplicationVersion,
  CustomerOrganization,
  DeploymentTarget,
  DeploymentWithLatestRevision,
} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown, faPlus} from '@fortawesome/free-solid-svg-icons';
import {
  catchError,
  combineLatest,
  combineLatestWith,
  debounceTime,
  distinctUntilChanged,
  first,
  map,
  Observable,
  of,
  shareReplay,
  switchMap,
} from 'rxjs';
import {arraysEqual, compareBy} from '../../util/arrays';
import {filteredByFormControl} from '../../util/filter';
import {refreshInBackground} from '../../util/refresh';
import {SecureImagePipe} from '../../util/secureImage';
import {MultiSelectComponent, MultiSelectOption} from '../components/multi-select/multi-select.component';
import {PageComponent} from '../components/page.component';
import {QuotaLimitComponent} from '../components/quota-limit.component';
import {SearchBarComponent} from '../components/search-bar.component';
import {ApplicationsService} from '../services/applications.service';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {DeploymentTargetsMetricsService} from '../services/deployment-target-metrics.service';
import {DeploymentTargetsService} from '../services/deployment-targets.service';
import {OrganizationService} from '../services/organization.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {DeploymentTargetLatestMetrics} from '../types/deployment-target-metrics';
import {DeploymentModalComponent} from './deployment-modal.component';
import {
  DeploymentDisplayStatus,
  deploymentDisplayStatus,
  deploymentDisplayStatusSelectOptions,
  isDeploymentDisplayStatus,
} from './deployment-target-card/deployment-status-text.component';
import {DeploymentTargetCardComponent} from './deployment-target-card/deployment-target-card.component';
import {DeploymentWizardComponent} from './deployment-wizard/deployment-wizard.component';

export interface DeploymentTargetViewData extends DeploymentTarget {
  metrics?: DeploymentTargetLatestMetrics;
}

export interface CustomerDeploymentTargets {
  customerOrganization?: CustomerOrganization;
  deploymentTargets: DeploymentTargetViewData[];
}

const localStoragerCollapsedCustomerIds = 'collapsedCustomerIds';
// customer organization ids are UUIDs, so this cannot collide with one
const internalCustomerOption = 'internal';
const deploymentDisplayStatusLabels = new Map(
  deploymentDisplayStatusSelectOptions.map((option) => [option.value, option.label])
);

@Component({
  selector: 'app-deployment-targets',
  imports: [
    AsyncPipe,
    FaIconComponent,
    DeploymentWizardComponent,
    OverlayModule,
    DeploymentTargetCardComponent,
    DeploymentModalComponent,
    SecureImagePipe,
    QuotaLimitComponent,
    PageComponent,
    SearchBarComponent,
    MultiSelectComponent,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './deployment-targets.component.html',
})
export class DeploymentTargetsComponent implements AfterViewInit {
  public readonly auth = inject(AuthService);
  private readonly overlay = inject(OverlayService);
  private readonly applications = inject(ApplicationsService);
  private readonly deploymentTargets = inject(DeploymentTargetsService);
  private readonly deploymentTargetMetrics = inject(DeploymentTargetsMetricsService);
  private readonly organizationService = inject(OrganizationService);
  private readonly customerOrganizations = inject(CustomerOrganizationsService);
  private readonly context = inject(ContextService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  protected readonly plusIcon = faPlus;
  protected readonly faChevronDown = faChevronDown;
  protected readonly collapsedCustomerIds = signal<string[]>(
    (() => {
      const s = localStorage.getItem(localStoragerCollapsedCustomerIds);
      if (s) {
        try {
          return JSON.parse(s);
        } catch (e) {
          console.warn(e);
        }
      }
      return [];
    })()
  );

  private modal?: DialogRef;

  protected readonly deploymentWizard = viewChild.required<TemplateRef<unknown>>('deploymentWizard');
  protected readonly deploymentModal = viewChild.required<TemplateRef<unknown>>('deploymentModal');

  selectedDeploymentTarget = signal<DeploymentTarget | undefined>(undefined);
  selectedDeployment = signal<DeploymentWithLatestRevision | undefined>(undefined);
  selectedApplicationVersionId = signal<string | undefined>(undefined);

  readonly filterForm = new FormGroup({
    search: new FormControl(this.route.snapshot.queryParamMap.get('search') ?? ''),
  });

  protected readonly canFilterByCustomer = this.auth.isVendor() || this.auth.isPartner();
  private readonly queryParams = toSignal(this.route.queryParamMap, {requireSync: true});
  // Equal by content, so that changing one filter does not restart the poll that depends on another.
  protected readonly selectedCustomerIds = computed(
    () => {
      const params = this.queryParams();
      return this.canFilterByCustomer
        ? [
            ...(params.get('internal') === 'true' ? [internalCustomerOption] : []),
            ...params.getAll('customerOrganizationId'),
          ]
        : [];
    },
    {equal: arraysEqual}
  );
  protected readonly selectedApplicationIds = computed(() => this.queryParams().getAll('applicationId'), {
    equal: arraysEqual,
  });
  protected readonly selectedStatuses = computed(
    () => this.queryParams().getAll('status').filter(isDeploymentDisplayStatus),
    {equal: arraysEqual}
  );
  protected readonly statusSelectOptions = deploymentDisplayStatusSelectOptions;

  protected readonly customerSelectOptions = toSignal(
    this.canFilterByCustomer
      ? this.customerOrganizations.getCustomerOrganizations().pipe(
          map((customers) => [
            // Only vendors have deployment targets that belong to no customer.
            ...(this.auth.isVendor() ? [{value: internalCustomerOption, label: 'Internal'}] : []),
            ...[...customers]
              .sort(compareBy((co) => co.name))
              .map<MultiSelectOption>((co) => ({value: co.id, label: co.name})),
          ]),
          catchError(() => of([]))
        )
      : of([]),
    {initialValue: []}
  );

  private readonly applications$ = this.applications.list();

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
    if (this.auth.isPartner() || (customerIds.length > 0 && !customerIds.includes(internalCustomerOption))) {
      return undefined;
    } else {
      return this.limit();
    }
  });
  protected readonly count = toSignal(
    combineLatest([this.deploymentTargets$, this.context.getUser()]).pipe(
      map(
        ([targets, user]) => targets.filter((it) => it.customerOrganization?.id === user.customerOrganizationId).length
      )
    ),
    {initialValue: 0}
  );
  protected readonly customerDeploymentTargetCounts = toSignal(
    this.deploymentTargets$.pipe(
      map((targets) =>
        targets.reduce<Record<string, number>>((counts, dt) => {
          if (dt.customerOrganization) {
            counts[dt.customerOrganization.id] = (counts[dt.customerOrganization.id] ?? 0) + 1;
          }
          return counts;
        }, {})
      )
    ),
    {initialValue: {} as Record<string, number>}
  );

  protected readonly filteredDeploymentTargets$: Observable<CustomerDeploymentTargets[]> = filteredByFormControl(
    this.deploymentTargets$,
    this.filterForm.controls.search,
    (dt, search) => {
      const query = search.toLowerCase();
      return [
        dt.name,
        dt.customerOrganization?.name ?? 'Internal',
        ...dt.deployments.flatMap((d) => [
          d.application.name,
          deploymentDisplayStatusLabels.get(deploymentDisplayStatus(d)),
        ]),
      ].some((it) => it?.toLowerCase().includes(query) ?? false);
    }
  ).pipe(
    combineLatestWith(toObservable(this.deploymentFilter)),
    map(([deploymentTargets, {applicationIds, statuses}]) =>
      applicationIds.length === 0 && statuses.length === 0
        ? deploymentTargets
        : deploymentTargets.filter((dt) =>
            dt.deployments.some(
              (d) =>
                (applicationIds.length === 0 ||
                  (d.application.id !== undefined && applicationIds.includes(d.application.id))) &&
                (statuses.length === 0 || statuses.includes(deploymentDisplayStatus(d)))
            )
          )
    ),
    combineLatestWith(this.deploymentTargetMetrics$),
    map(([deploymentTargets, deploymentTargetMetrics]) =>
      deploymentTargets.map((dt) => ({
        ...dt,
        metrics: deploymentTargetMetrics.find((x) => x.deploymentTargetId === dt.id),
      }))
    ),
    map((deploymentTargets) =>
      // For vendors and partners: group deployment targets by customer organization
      // For customers: just put all deployment targets into one group
      (this.auth.isVendor() || this.auth.isPartner()
        ? [
            ...(this.auth.isVendor()
              ? [{deploymentTargets: deploymentTargets.filter((it) => it.customerOrganization === undefined)}]
              : []),
            ...Object.values(
              deploymentTargets
                .map((dt) => dt.customerOrganization)
                .filter((co) => co !== undefined)
                .sort(compareBy((co) => co.name))
                .reduce<Record<string, CustomerOrganization>>((agg, co) => ({...agg, [co.id]: co}), {})
            ).map<CustomerDeploymentTargets>((customerOrganization) => ({
              customerOrganization,
              deploymentTargets: deploymentTargets.filter(
                (dt) => dt.customerOrganization?.id === customerOrganization.id
              ),
            })),
          ]
        : [{deploymentTargets}]
      ).filter((it) => it.deploymentTargets.length > 0)
    )
  );

  constructor() {
    refreshInBackground(this.applications.refresh());
    const search = this.filterForm.controls.search;
    // The component is reused when only the query changes, e.g. when the sidebar links to the page it is on.
    this.route.queryParamMap
      .pipe(
        map((params) => params.get('search') ?? ''),
        distinctUntilChanged(),
        takeUntilDestroyed()
      )
      .subscribe((value) => {
        if (value !== search.value) {
          search.setValue(value);
        }
      });
    search.valueChanges
      .pipe(debounceTime(300), takeUntilDestroyed())
      .subscribe((value) => this.setQueryParams({search: value || null}));
    effect(() => localStorage.setItem(localStoragerCollapsedCustomerIds, JSON.stringify(this.collapsedCustomerIds())));
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

  protected selectStatuses(statuses: DeploymentDisplayStatus[]) {
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

  protected toggleCustomerCollapsed(customerId: string): void {
    this.collapsedCustomerIds.update((prev) =>
      prev.includes(customerId) ? prev.filter((el) => el !== customerId) : prev.concat(customerId)
    );
  }
}
