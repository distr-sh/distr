import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe, DecimalPipe} from '@angular/common';
import {Component, computed, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {CustomerOrganization, CustomerOrganizationFeature, CustomerOrganizationWithUsage} from '@distr-sh/distr-sdk';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {
  faBuildingUser,
  faChevronDown,
  faCircleExclamation,
  faEdit,
  faMagnifyingGlass,
  faPlus,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, filter, firstValueFrom, map, startWith, Subject, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {SecureImagePipe} from '../../../util/secureImage';
import {RequireVendorDirective} from '../../directives/required-role.directive';
import {ArtifactLicensesService} from '../../services/artifact-licenses.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {ImageUploadService} from '../../services/image-upload.service';
import {LicensesService} from '../../services/licenses.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {QuotaLimitComponent} from '../quota-limit.component';

@Component({
  templateUrl: './customer-organizations.component.html',
  imports: [
    ReactiveFormsModule,
    FontAwesomeModule,
    DatePipe,
    SecureImagePipe,
    AsyncPipe,
    DecimalPipe,
    RouterLink,
    RequireVendorDirective,
    QuotaLimitComponent,
    OverlayModule,
  ],
})
export class CustomerOrganizationsComponent {
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPlus = faPlus;
  protected readonly faBuildingUser = faBuildingUser;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faEdit = faEdit;
  protected readonly faChevronDown = faChevronDown;

  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly toast = inject(ToastService);
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly organizationService = inject(OrganizationService);
  private readonly artifactLicensesService = inject(ArtifactLicensesService);
  private readonly licensesService = inject(LicensesService);
  protected readonly featureFlags = inject(FeatureFlagService);
  protected readonly auth = inject(AuthService);

  private readonly organization = toSignal(this.organizationService.get());
  protected readonly limit = computed(() => this.organization()?.subscriptionCustomerOrganizationQuantity);

  protected readonly filterForm = this.fb.group({
    search: this.fb.control(''),
  });
  private readonly refresh$ = new Subject<void>();
  protected readonly customerOrganizations = toSignal(
    combineLatest([
      this.filterForm.valueChanges.pipe(
        map((filter) => filter.search ?? ''),
        startWith('')
      ),
      this.refresh$.pipe(
        startWith(undefined),
        switchMap(() => this.customerOrganizationsService.getCustomerOrganizations())
      ),
    ]).pipe(
      map(([filter, organizations]) =>
        filter.length > 0
          ? organizations.filter((organization) => organization.name.toLowerCase().includes(filter.toLowerCase()))
          : organizations
      )
    )
  );

  private readonly createCustomerDialog = viewChild.required<TemplateRef<unknown>>('createCustomerDialog');
  private modalRef?: DialogRef;
  protected readonly createForm = this.fb.group({
    id: this.fb.control(''),
    name: this.fb.control('', [Validators.required]),
    imageId: this.fb.control(''),
  });
  protected createFormLoading = false;

  protected readonly allCustomerFeatures: readonly CustomerOrganizationFeature[] = [
    'deployment_targets',
    'alerts',
    'artifacts',
  ];

  protected readonly openCustomerFeaturesDropdownId = signal<string | void>(undefined);
  protected readonly openCustomerFeaturesDropdownCustomer = computed(() => {
    const id = this.openCustomerFeaturesDropdownId();
    return id ? this.customerOrganizations()?.find((it) => it.id === id) : undefined;
  });
  protected dropdownWidth = 0;

  protected showCreateDialog() {
    this.closeCreateDialog();
    this.modalRef = this.overlay.showModal(this.createCustomerDialog());
  }

  protected showUpdateDialog(value: CustomerOrganization) {
    this.closeCreateDialog();
    this.createForm.patchValue(value);
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

    this.createFormLoading = true;

    const request = {
      name: this.createForm.value.name!,
      imageId: this.createForm.value.imageId || undefined,
    };

    try {
      if (this.createForm.value.id) {
        await firstValueFrom(
          this.customerOrganizationsService.updateCustomerOrganization(this.createForm.value.id, request)
        );
      } else {
        await firstValueFrom(this.customerOrganizationsService.createCustomerOrganization(request));
      }

      this.closeCreateDialog();
      this.refresh$.next();
    } finally {
      this.createFormLoading = false;
    }
  }

  protected async uploadImage(value: CustomerOrganization): Promise<void> {
    const imageId = await firstValueFrom(this.imageUploadService.showDialog({scope: 'platform'}));
    if (!imageId || imageId === value.imageId) {
      return;
    }
    await firstValueFrom(
      this.customerOrganizationsService.updateCustomerOrganization(value.id, {name: value.name, imageId})
    );
    this.refresh$.next();
  }

  protected delete(target: CustomerOrganizationWithUsage): void {
    this.overlay
      .confirm({
        message: {
          message: 'Are you sure you want to delete this customer?',
          alert:
            target.userCount > 0 || target.deploymentTargetCount > 0
              ? {
                  type: 'warning',
                  message: `Deleting this customer will also delete its associated users (${target.userCount}) and deployment targets (${target.deploymentTargetCount}) from your organization.`,
                }
              : undefined,
        },
        requiredConfirmInputText: target.name,
      })
      .pipe(
        filter((it) => it === true),
        switchMap(() => this.customerOrganizationsService.deleteCustomerOrganization(target.id!))
      )
      .subscribe({
        next: () => {
          this.refresh$.next();
          this.artifactLicensesService.refresh();
          this.licensesService.refresh();
        },
        error: (e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
  }

  protected getFeatureLabel(feature: CustomerOrganizationFeature): string {
    switch (feature) {
      case 'deployment_targets':
        return 'Deployments';
      case 'artifacts':
        return 'Artifacts';
      case 'alerts':
        return 'Alerts';
      default:
        return feature;
    }
  }

  protected isFeatureIndent(feature: CustomerOrganizationFeature): boolean {
    return feature === 'alerts';
  }

  protected async toggleFeature(customer: CustomerOrganization, feature: CustomerOrganizationFeature) {
    const featureSet = new Set(customer.features);
    if (featureSet.has(feature)) {
      featureSet.delete(feature);
      if (feature === 'deployment_targets') {
        featureSet.delete('alerts');
      }
    } else {
      featureSet.add(feature);
      if (feature === 'alerts') {
        featureSet.add('deployment_targets');
      }
    }

    try {
      await firstValueFrom(
        this.customerOrganizationsService.updateCustomerOrganization(customer.id, {
          ...customer,
          features: Array.from(featureSet),
        })
      );
      this.toast.success('Customer features updated');
      this.refresh$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected showCustomerFeaturesDropdown(customer: CustomerOrganization, btn: HTMLButtonElement) {
    this.dropdownWidth = btn.getBoundingClientRect().width;
    this.openCustomerFeaturesDropdownId.set(customer.id);
  }

  protected hideCustomerFeaturesDropdown(): void {
    this.openCustomerFeaturesDropdownId.set(undefined);
  }
}
