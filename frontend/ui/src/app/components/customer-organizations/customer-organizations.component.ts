import {AsyncPipe, NgPlural, NgPluralCase, NgTemplateOutlet} from '@angular/common';
import {Component, computed, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faPlus, faXmark} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, firstValueFrom, of, switchMap} from 'rxjs';
import {refreshInBackground} from '../../../util/refresh';
import {ClickableRowDirective} from '../../directives/clickable-row.directive';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ApplicationsService} from '../../services/applications.service';
import {ArtifactPullsService} from '../../services/artifact-pulls.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {LicenseTemplatesService} from '../../services/license-templates.service';
import {LicensesService} from '../../services/licenses.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {AlertComponent} from '../alert/alert.component';
import {AvatarComponent} from '../avatar.component';
import {PageComponent} from '../page.component';
import {QuotaLimitComponent} from '../quota-limit.component';
import {SearchBarComponent} from '../search-bar.component';
import {CustomerLicenseBadgeComponent} from './customer-license-badge.component';
import {toCustomerListRows} from './customer-list-mappers';
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
  private readonly licenseTemplatesService = inject(LicenseTemplatesService);
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly organizationService = inject(OrganizationService);
  protected readonly featureFlags = inject(FeatureFlagService);
  protected readonly auth = inject(AuthService);
  private readonly isVendor = this.auth.isVendor();

  private readonly organization = toSignal(this.organizationService.get());
  protected readonly limit = computed(() => this.organization()?.subscriptionCustomerOrganizationQuantity);
  protected readonly currentCustomerOrganizationCount = computed(
    () => this.organization()?.currentCustomerOrganizationCount
  );

  protected readonly filterForm = this.fb.group({
    search: this.fb.control(''),
  });
  private readonly search = toSignal(this.filterForm.controls.search.valueChanges, {initialValue: ''});
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
  private readonly licenseSources = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(
      switchMap((enabled) =>
        enabled
          ? combineLatest({
              licenses: this.licensesService.list(),
              // Templates only exist with vendor billing, and partners cannot read them.
              templates: this.isVendor
                ? this.featureFlags.isVendorBillingEnabled$.pipe(
                    switchMap((billing) => (billing ? this.licenseTemplatesService.list() : of([])))
                  )
                : of([]),
            })
          : of(null)
      )
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
      this.licenseSources() ?? undefined,
      this.supportBundles() ?? undefined
    );
  });

  protected readonly rows = computed(() => {
    const search = this.search().toLowerCase();
    const rows = this.allRows();
    return search.length > 0 ? rows?.filter((row) => row.name.toLowerCase().includes(search)) : rows;
  });

  protected readonly showLicenseColumn = toSignal(this.featureFlags.isLicensingEnabled$, {initialValue: false});

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
      ),
      firstValueFrom(this.featureFlags.isVendorBillingEnabled$).then((enabled) =>
        enabled && this.isVendor ? this.licenseTemplatesService.refresh() : undefined
      )
    );
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
