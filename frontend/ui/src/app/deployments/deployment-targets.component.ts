import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {AfterViewInit, Component, computed, inject, signal, TemplateRef, ViewChild} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, FormsModule, ReactiveFormsModule} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {
  ApplicationVersion,
  CustomerOrganization,
  DeploymentTarget,
  DeploymentWithLatestRevision,
} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBullhorn, faLightbulb, faMagnifyingGlass, faPlus} from '@fortawesome/free-solid-svg-icons';
import {catchError, combineLatest, combineLatestWith, first, map, Observable, of} from 'rxjs';
import {compareBy} from '../../util/arrays';
import {filteredByFormControl} from '../../util/filter';
import {SecureImagePipe} from '../../util/secureImage';
import {drawerFlyInOut} from '../animations/drawer';
import {modalFlyInOut} from '../animations/modal';
import {QuotaLimitComponent} from '../components/quota-limit.component';
import {ApplicationsService} from '../services/applications.service';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {
  DeploymentTargetLatestMetrics,
  DeploymentTargetsMetricsService,
} from '../services/deployment-target-metrics.service';
import {DeploymentTargetsService} from '../services/deployment-targets.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {OrganizationService} from '../services/organization.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {DeploymentModalComponent} from './deployment-modal.component';
import {DeploymentTargetCardComponent} from './deployment-target-card/deployment-target-card.component';
import {DeploymentWizardComponent} from './deployment-wizard/deployment-wizard.component';

type DeploymentWithNewerVersion = {dt: DeploymentTarget; d: DeploymentWithLatestRevision; version: ApplicationVersion};

export interface DeploymentTargetViewData extends DeploymentTarget {
  metrics?: DeploymentTargetLatestMetrics;
}

export interface CustomerDeploymentTargets {
  customerOrganization?: CustomerOrganization;
  deploymentTargets: DeploymentTargetViewData[];
}

@Component({
  selector: 'app-deployment-targets',
  imports: [
    AsyncPipe,
    FaIconComponent,
    FormsModule,
    ReactiveFormsModule,
    DeploymentWizardComponent,
    OverlayModule,
    DeploymentTargetCardComponent,
    DeploymentModalComponent,
    SecureImagePipe,
    QuotaLimitComponent,
    RouterLink,
  ],
  templateUrl: './deployment-targets.component.html',
  standalone: true,
  animations: [modalFlyInOut, drawerFlyInOut],
})
export class DeploymentTargetsComponent implements AfterViewInit {
  public readonly auth = inject(AuthService);
  private readonly overlay = inject(OverlayService);
  private readonly applications = inject(ApplicationsService);
  private readonly deploymentTargets = inject(DeploymentTargetsService);
  private readonly deploymentTargetMetrics = inject(DeploymentTargetsMetricsService);
  private readonly organizationService = inject(OrganizationService);
  private readonly context = inject(ContextService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly route = inject(ActivatedRoute);

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly plusIcon = faPlus;
  protected readonly faLightbulb = faLightbulb;
  protected readonly faBullhorn = faBullhorn;

  protected readonly isAlertsVisible = toSignal(
    this.featureFlags.isNotificationsEnabled$.pipe(
      combineLatestWith(this.context.getCustomerOrganization()),
      map(
        ([enabled, customerOrg]) =>
          enabled && this.auth.isCustomer() && (customerOrg?.features?.includes('notifications') ?? false)
      )
    ),
    {initialValue: false}
  );

  private modal?: DialogRef;

  @ViewChild('deploymentWizard') protected readonly deploymentWizard!: TemplateRef<unknown>;
  @ViewChild('deploymentModal') protected readonly deploymentModal!: TemplateRef<unknown>;

  selectedDeploymentTarget = signal<DeploymentTarget | undefined>(undefined);
  selectedDeployment = signal<DeploymentWithLatestRevision | undefined>(undefined);
  selectedApplicationVersionId = signal<string | undefined>(undefined);

  readonly filterForm = new FormGroup({
    search: new FormControl(this.route.snapshot.queryParamMap.get('search') ?? ''),
  });

  readonly deploymentTargets$ = this.deploymentTargets.poll().pipe(takeUntilDestroyed());
  readonly deploymentTargetMetrics$ = this.deploymentTargetMetrics.poll().pipe(
    takeUntilDestroyed(),
    catchError(() => of([]))
  );

  private readonly organization = toSignal(this.organizationService.get());
  protected readonly limit = computed(() => {
    const org = this.organization();
    return !(org && org.subscriptionLimits) ? undefined : org.subscriptionLimits.maxDeploymentsPerCustomerOrganization;
  });
  protected readonly count = toSignal(
    combineLatest([this.deploymentTargets$, this.context.getUser()]).pipe(
      map(
        ([targets, user]) => targets.filter((it) => it.customerOrganization?.id === user.customerOrganizationId).length
      )
    ),
    {initialValue: 0}
  );

  protected readonly filteredDeploymentTargets$: Observable<CustomerDeploymentTargets[]> = filteredByFormControl(
    this.deploymentTargets$,
    this.filterForm.controls.search,
    (dt, search) =>
      !search ||
      [dt.name, dt.customerOrganization?.name].some((it) => it?.toLowerCase()?.includes(search.toLowerCase()) ?? false)
  ).pipe(
    combineLatestWith(this.deploymentTargetMetrics$),
    map(([deploymentTargets, deploymentTargetMetrics]) =>
      deploymentTargets.map((dt) => ({
        ...dt,
        metrics: deploymentTargetMetrics.find((x) => x.id === dt.id),
      }))
    ),
    map((deploymentTargets) =>
      // For vendors: group deployment targets by customer organization
      // For customers: just put all deployment targets into one group
      (this.auth.isVendor()
        ? [
            {deploymentTargets: deploymentTargets.filter((it) => it.customerOrganization === undefined)},
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

  private readonly applications$ = this.applications.list();

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
    this.modal = this.overlay.showModal(this.deploymentModal, {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  protected openWizard() {
    this.hideModal();
    this.modal = this.overlay.showModal(this.deploymentWizard, {
      hasBackdrop: true,
      backdropStyleOnly: true,
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  protected hideModal(): void {
    this.modal?.close();
  }
}
