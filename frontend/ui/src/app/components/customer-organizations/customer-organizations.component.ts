import {AsyncPipe, NgPlural, NgPluralCase, NgTemplateOutlet} from '@angular/common';
import {Component, computed, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, Params, Router} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faPlus, faXmark} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, firstValueFrom, of, switchMap} from 'rxjs';
import {arraysEqual} from '../../../util/arrays';
import {syncSearchWithQueryParam} from '../../../util/filter';
import {refreshInBackground} from '../../../util/refresh';
import {ClickableRowDirective} from '../../directives/clickable-row.directive';
import {customerLicenseStatusSelectOptions, isCustomerLicenseStatus} from '../../licenses/customer-license-status';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ApplicationsService} from '../../services/applications.service';
import {ArtifactPullsService} from '../../services/artifact-pulls.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {LicensesService} from '../../services/licenses.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {AlertComponent} from '../alert/alert.component';
import {AvatarComponent} from '../avatar.component';
import {MultiSelectComponent} from '../multi-select/multi-select.component';
import {PageComponent} from '../page.component';
import {QuotaLimitComponent} from '../quota-limit.component';
import {SearchBarComponent} from '../search-bar.component';
import {CustomerLicenseBadgeComponent} from './customer-license-badge.component';
import {CustomerUpdateStatus, customerUpdateStatusSelectOptions, isCustomerUpdateStatus} from './customer-list-display';
import {CustomerLicenseStatus, toCustomerListRows} from './customer-list-mappers';
import {CustomerDetailTabId, customerDetailTabs} from './detail/customer-detail-navigation';

@Component({
  templateUrl: './customer-organizations.component.html',
  imports: [
    ReactiveFormsModule,
    FontAwesomeModule,
    AsyncPipe,
    NgPlural,
    NgPluralCase,
    NgTemplateOutlet,
    ClickableRowDirective,
    QuotaLimitComponent,
    AvatarComponent,
    PageComponent,
    SearchBarComponent,
    MultiSelectComponent,
    AlertComponent,
    CustomerLicenseBadgeComponent,
  ],
})
export class CustomerOrganizationsComponent {
  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;
  protected readonly columnTabs = {
    customer: 'overview',
    license: 'licenses',
    support: 'support',
    deployments: 'deployments',
  } satisfies Record<string, CustomerDetailTabId>;
  protected readonly tabTitles = Object.fromEntries(
    customerDetailTabs.ids.map((id) => [id, `Open ${customerDetailTabs.label(id)}`])
  );

  private readonly router = inject(Router);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly artifactPullsService = inject(ArtifactPullsService);
  private readonly licensesService = inject(LicensesService);
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly organizationService = inject(OrganizationService);
  protected readonly featureFlags = inject(FeatureFlagService);
  protected readonly auth = inject(AuthService);
  private readonly organization = toSignal(this.organizationService.get());
  protected readonly limit = computed(() => this.organization()?.subscriptionCustomerOrganizationQuantity);
  protected readonly currentCustomerOrganizationCount = computed(
    () => this.organization()?.currentCustomerOrganizationCount
  );

  private readonly route = inject(ActivatedRoute);
  private readonly queryParams = toSignal(this.route.queryParamMap, {requireSync: true});
  protected readonly filterForm = this.fb.group({
    search: this.fb.control(this.route.snapshot.queryParamMap.get('search') ?? ''),
  });
  private readonly search = toSignal(this.filterForm.controls.search.valueChanges, {
    initialValue: this.filterForm.controls.search.value,
  });
  protected readonly licenseStatusSelectOptions = customerLicenseStatusSelectOptions;
  protected readonly selectedLicenseStatuses = computed(
    () => this.queryParams().getAll('license').filter(isCustomerLicenseStatus),
    {equal: arraysEqual}
  );
  protected readonly updateStatusSelectOptions = customerUpdateStatusSelectOptions;
  protected readonly selectedUpdateStatuses = computed(
    () => this.queryParams().getAll('updates').filter(isCustomerUpdateStatus),
    {equal: arraysEqual}
  );
  private readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());

  private readonly deploymentSources = toSignal(
    combineLatest({
      targets: this.deploymentTargetsService.list(),
      applications: this.applicationsService.list(),
      entitlements: this.featureFlags.isLicensingEnabled$.pipe(
        switchMap((enabled) => (enabled ? this.applicationEntitlementsService.list() : of([])))
      ),
      pulls: this.artifactPullsService.listLatest(),
    })
  );

  /** `null` while the organization has no licensing. */
  private readonly licenses = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(
      switchMap((enabled) => (enabled ? this.licensesService.list() : of(null)))
    )
  );

  /** `null` while the organization's plan has no support bundles. */
  private readonly supportBundles = toSignal(
    this.featureFlags.isSupportBundlesEnabled$.pipe(
      switchMap((enabled) => (enabled ? this.supportBundlesService.list() : of(null)))
    )
  );
  protected readonly showSupportColumn = toSignal(this.featureFlags.isSupportBundlesEnabled$, {initialValue: false});

  private readonly allRows = computed(() => {
    const customers = this.customerOrganizations();
    if (customers === undefined) {
      return undefined;
    }
    return toCustomerListRows(
      customers,
      this.deploymentSources(),
      this.licenses() ?? undefined,
      this.supportBundles() ?? undefined
    );
  });

  /** Undefined until every column a filter is set for has loaded. */
  protected readonly rows = computed(() => {
    const rows = this.allRows();
    const search = this.search().toLowerCase();
    const licenses = this.licenses() === null ? [] : this.selectedLicenseStatuses();
    const updates = this.selectedUpdateStatuses();
    if (
      rows === undefined ||
      (licenses.length > 0 && this.licenses() === undefined) ||
      (updates.length > 0 && this.deploymentSources() === undefined)
    ) {
      return undefined;
    }
    return rows.filter(
      (row) =>
        (search.length === 0 || row.name.toLowerCase().includes(search)) &&
        (licenses.length === 0 || (row.license !== undefined && licenses.includes(row.license))) &&
        (updates.length === 0 ||
          (row.isOutdated !== undefined && updates.includes(row.isOutdated ? 'outdated' : 'up_to_date')))
    );
  });

  protected readonly showLicenseColumn = toSignal(this.featureFlags.isLicensingEnabled$, {initialValue: false});

  protected readonly columnCount = computed(
    () => 3 + (this.showLicenseColumn() ? 1 : 0) + (this.showSupportColumn() ? 1 : 0)
  );
  protected readonly placeholderRows = [0, 1, 2, 3, 4];

  private readonly createCustomerDialog = viewChild.required<TemplateRef<unknown>>('createCustomerDialog');
  private modalRef?: DialogRef;
  protected readonly createForm = this.fb.group({
    name: this.fb.control('', [Validators.required]),
  });
  protected readonly createFormLoading = signal(false);

  constructor() {
    refreshInBackground(
      this.customerOrganizationsService.refresh(),
      this.deploymentTargetsService.refresh(),
      this.applicationsService.refresh(),
      this.artifactPullsService.refreshLatest(),
      firstValueFrom(this.featureFlags.isLicensingEnabled$).then((enabled) =>
        enabled ? this.applicationEntitlementsService.refresh() : undefined
      )
    );
    syncSearchWithQueryParam(this.filterForm.controls.search);
  }

  protected selectLicenseStatuses(statuses: CustomerLicenseStatus[]) {
    this.setQueryParams({license: statuses.length > 0 ? statuses : null});
  }

  protected selectUpdateStatuses(statuses: CustomerUpdateStatus[]) {
    this.setQueryParams({updates: statuses.length > 0 ? statuses : null});
  }

  private setQueryParams(queryParams: Params) {
    this.router.navigate([], {relativeTo: this.route, queryParams, queryParamsHandling: 'merge', replaceUrl: true});
  }

  protected showCreateDialog() {
    this.closeCreateDialog();
    this.modalRef = this.overlay.showModal(this.createCustomerDialog());
  }

  protected closeCreateDialog(reset: boolean = true): void {
    this.modalRef?.close();

    if (reset) {
      this.createForm.reset();
    }
  }

  protected async submitCreateForm() {
    this.createForm.markAllAsTouched();

    if (this.createForm.invalid) {
      return;
    }

    this.createFormLoading.set(true);

    try {
      const customer = await firstValueFrom(
        this.customerOrganizationsService.createCustomerOrganization({
          name: this.createForm.value.name!,
        })
      );

      this.closeCreateDialog();
      await this.router.navigate(['/', 'customers', customer.id]);
    } finally {
      this.createFormLoading.set(false);
    }
  }
}
