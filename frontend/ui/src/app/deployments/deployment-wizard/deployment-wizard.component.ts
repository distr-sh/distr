import {CdkStep, CdkStepper} from '@angular/cdk/stepper';
import {AsyncPipe} from '@angular/common';
import {Component, computed, DestroyRef, effect, inject, OnInit, output, signal, viewChild} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {
  Application,
  CustomerOrganization,
  DeploymentTarget,
  DeploymentTargetScope,
  DeploymentType,
} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faDocker} from '@fortawesome/free-brands-svg-icons';
import {faBuildingUser, faCheckCircle, faDharmachakra, faShip, faXmark} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, distinctUntilChanged, firstValueFrom, map, of, switchMap, take} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {SecureImagePipe} from '../../../util/secureImage';
import {
  KUBERNETES_RESOURCE_MAX_LENGTH,
  KUBERNETES_RESOURCE_NAME_REGEX,
  RESOURCE_QUANTITY_REGEX,
} from '../../../util/validation';
import {modalFlyInOut} from '../../animations/modal';
import {ConnectInstructionsComponent} from '../../components/connect-instructions/connect-instructions.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ApplicationsService} from '../../services/applications.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {OrganizationService} from '../../services/organization.service';
import {ToastService} from '../../services/toast.service';
import {DeploymentFormComponent, mapToDeploymentRequest} from '../deployment-form/deployment-form.component';
import {DeploymentWizardStepperComponent} from './deployment-wizard-stepper.component';

@Component({
  selector: 'app-deployment-wizard',
  templateUrl: './deployment-wizard.component.html',
  imports: [
    AsyncPipe,
    ReactiveFormsModule,
    FaIconComponent,
    DeploymentWizardStepperComponent,
    CdkStep,
    ConnectInstructionsComponent,
    AutotrimDirective,
    SecureImagePipe,
    DeploymentFormComponent,
  ],
  animations: [modalFlyInOut],
})
export class DeploymentWizardComponent implements OnInit {
  protected readonly faXmark = faXmark;
  protected readonly faShip = faShip;
  protected readonly faDocker = faDocker;
  protected readonly faDharmachakra = faDharmachakra;
  protected readonly faBuildingUser = faBuildingUser;
  protected readonly faCheckCircle = faCheckCircle;

  private readonly toast = inject(ToastService);
  private readonly applications = inject(ApplicationsService);
  private readonly deploymentTargets = inject(DeploymentTargetsService);
  private readonly customerOrganizations = inject(CustomerOrganizationsService);
  private readonly applicationEntitlements = inject(ApplicationEntitlementsService);
  private readonly organization = inject(OrganizationService);
  private readonly organizationBranding = inject(OrganizationBrandingService);
  private readonly fb = inject(FormBuilder);
  private readonly fbnn = this.fb.nonNullable;
  protected readonly auth = inject(AuthService);
  protected readonly featureFlags = inject(FeatureFlagService);

  private readonly stepper = viewChild<CdkStepper>('stepper');

  readonly closed = output<void>();

  // Step 1: Customer Selection (optional)
  readonly customerForm = this.fbnn.group({
    internal: this.fbnn.control<boolean>(false),
    customerOrganizationId: this.fbnn.control<string | undefined>(undefined),
  });

  // Step 2: Application Selection
  readonly applicationForm = this.fbnn.group({
    applicationId: this.fbnn.control<string | undefined>(undefined, Validators.required),
  });

  // Step 3: Deployment Target Configuration
  readonly deploymentTargetForm = new FormGroup({
    name: new FormControl<string>('', Validators.required),
    namespace: new FormControl<string>('', [
      Validators.required,
      Validators.maxLength(KUBERNETES_RESOURCE_MAX_LENGTH),
      Validators.pattern(KUBERNETES_RESOURCE_NAME_REGEX),
    ]),
    autohealEnabled: new FormControl<boolean>(true, {nonNullable: true}),
    clusterScope: new FormControl<boolean>(true, {nonNullable: true}),
    imageCleanupEnabled: new FormControl<boolean>(true, {nonNullable: true}),
    customResources: new FormControl<boolean>(false, {nonNullable: true}),
    resources: new FormGroup({
      cpuRequest: new FormControl<string>('100m', {
        nonNullable: true,
        validators: [Validators.required, Validators.pattern(RESOURCE_QUANTITY_REGEX)],
      }),
      memoryRequest: new FormControl<string>('256Mi', {
        nonNullable: true,
        validators: [Validators.required, Validators.pattern(RESOURCE_QUANTITY_REGEX)],
      }),
      cpuLimit: new FormControl<string>('1', {
        nonNullable: true,
        validators: [Validators.required, Validators.pattern(RESOURCE_QUANTITY_REGEX)],
      }),
      memoryLimit: new FormControl<string>('256Mi', {
        nonNullable: true,
        validators: [Validators.required, Validators.pattern(RESOURCE_QUANTITY_REGEX)],
      }),
    }),
    scope: new FormControl<DeploymentTargetScope>('cluster', {nonNullable: true}),
  });

  // Step 4: Application Configuration
  readonly applicationConfigForm = new FormGroup({
    deploymentFormData: new FormControl<any>(null, Validators.required),
  });

  // Step 5: Connect Instructions (no form, just display)
  readonly connectForm = new FormGroup({});

  // State management
  protected readonly customerOrganizations$ = this.auth.isVendor()
    ? this.customerOrganizations.getCustomerOrganizations()
    : of([]);

  protected readonly applications$ = this.applications.list();
  protected readonly allApplicationEntitlements$ = this.featureFlags.isLicensingEnabled$.pipe(
    switchMap((enabled) => (enabled ? this.applicationEntitlements.list() : of([])))
  );
  protected readonly currentOrganization$ = this.organization.get();
  protected readonly vendorBranding$ = this.organizationBranding.get();
  protected readonly selectedApplication = signal<Application | undefined>(undefined);
  protected readonly selectedCustomerOrganizationId = toSignal(
    this.customerForm.controls.customerOrganizationId.valueChanges
  );
  protected readonly selectedDeploymentTarget = signal<DeploymentTarget | undefined>(undefined);
  protected readonly internalDeploymentCount = toSignal(
    this.deploymentTargets
      .list()
      .pipe(map((targets) => targets.filter((it) => it.customerOrganization === undefined).length)),
    {initialValue: 0}
  );

  // Filter applications based on customer entitlements
  protected readonly filteredApplications$ = combineLatest([
    this.applications$,
    this.allApplicationEntitlements$,
    toObservable(this.selectedCustomerOrganizationId),
  ]).pipe(
    map(([applications, entitlements, customerOrgId]) => {
      // If no customer is selected or no entitlement, show all applications
      if (!customerOrgId || entitlements.length === 0) {
        return applications;
      }

      // Filter applications to only show those with licenses for the selected customer
      const customerEntitlements = entitlements.filter((l) => l.customerOrganizationId === customerOrgId);
      const entitledApplicationIds = new Set(customerEntitlements.map((l) => l.applicationId));

      return applications.filter((app) => entitledApplicationIds.has(app.id));
    })
  );

  // Computed properties
  protected readonly showCustomerStep = computed(() => {
    return this.auth.isVendor();
  });

  protected readonly selectedDeploymentType = computed<DeploymentType>(() => {
    const app = this.selectedApplication();
    return app?.type ?? 'docker';
  });

  // Initial data for deployment form
  protected readonly deploymentFormInitialData = computed(() => {
    const app = this.selectedApplication();
    if (!app) {
      return null;
    }
    return {
      applicationId: app.id!,
    };
  });

  private readonly entitlementControlVisible$ = combineLatest([
    this.allApplicationEntitlements$.pipe(
      map((entitlements) => entitlements.length > 0),
      distinctUntilChanged()
    ),
    toObservable(this.selectedCustomerOrganizationId).pipe(
      map((id) => id !== ''),
      distinctUntilChanged()
    ),
  ]).pipe(map(([hasEntitlement, isCustomerOrganizationIdSet]) => hasEntitlement && isCustomerOrganizationIdSet));

  protected readonly entitlementControlVisible = toSignal(this.entitlementControlVisible$, {initialValue: false});

  protected readonly isApplicationConfigStep = computed(() => {
    const stepIndex = this.stepper()?.selectedIndex ?? 0;
    const adjustedIndex = this.showCustomerStep() ? stepIndex : stepIndex + 1;
    return adjustedIndex === 3;
  });

  protected getVendorLogoUrl(branding: {logo?: string; logoContentType?: string} | null): string {
    if (branding?.logo && branding?.logoContentType) {
      return `data:${branding.logoContentType};base64,${branding.logo}`;
    }
    return '/distr-logo.svg';
  }

  private loading = false;
  private readonly destroyRef = inject(DestroyRef);

  constructor() {
    this.applications.refresh();
    // Initialize deployment form with initial data reactively
    effect(() => {
      const initialData = this.deploymentFormInitialData();
      if (initialData) {
        this.applicationConfigForm.controls.deploymentFormData.patchValue(initialData);
      }
    });

    // Reset application form when customer organization changes
    // This prevents a vendor from accidentally creating a deployment with an application that a customer should have no access to
    effect(() => {
      this.selectedCustomerOrganizationId();
      this.applicationForm.reset();
    });
  }

  ngOnInit() {
    // If user is a customer, set selectedCustomerOrganizationId from organization
    if (!this.auth.isVendor()) {
      this.currentOrganization$
        .pipe(take(1))
        .subscribe((org) => this.customerForm.controls.customerOrganizationId.setValue(org.customerOrganizationId!));
    }

    // Watch application selection
    this.applicationForm.controls.applicationId.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((appId) => {
        firstValueFrom(this.applications$).then((apps) => {
          const app = apps.find((a) => a.id === appId);
          this.selectedApplication.set(app);

          // Reset deployment form data when application changes
          this.applicationConfigForm.controls.deploymentFormData.reset();

          // Enable/disable configuration form controls based on deployment type
          this.updateConfigurationFormControls(app?.type);
        });
      });

    // Watch cluster scope changes
    this.deploymentTargetForm.controls.clusterScope.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((value) => {
        this.deploymentTargetForm.controls.scope.setValue(value ? 'cluster' : 'namespace');
      });

    this.deploymentTargetForm.controls.customResources.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe((value) => {
        if (value) {
          this.deploymentTargetForm.controls.resources.enable();
        } else {
          this.deploymentTargetForm.controls.resources.disable();
        }
      });
  }

  private updateConfigurationFormControls(type: DeploymentType | undefined) {
    if (type === 'kubernetes') {
      this.deploymentTargetForm.controls.autohealEnabled.disable();
      this.deploymentTargetForm.controls.namespace.enable();
      this.deploymentTargetForm.controls.clusterScope.enable();
      this.deploymentTargetForm.controls.scope.enable();
      this.deploymentTargetForm.controls.imageCleanupEnabled.disable();
      this.deploymentTargetForm.controls.customResources.enable();
      if (this.deploymentTargetForm.controls.customResources.value) {
        this.deploymentTargetForm.controls.resources.enable();
      } else {
        this.deploymentTargetForm.controls.resources.disable();
      }
    } else if (type === 'docker') {
      this.deploymentTargetForm.controls.autohealEnabled.enable();
      this.deploymentTargetForm.controls.namespace.disable();
      this.deploymentTargetForm.controls.clusterScope.disable();
      this.deploymentTargetForm.controls.scope.disable();
      this.deploymentTargetForm.controls.imageCleanupEnabled.enable();
      this.deploymentTargetForm.controls.customResources.disable();
      this.deploymentTargetForm.controls.resources.disable();
    }
  }

  async attemptContinue() {
    if (this.loading) {
      return;
    }

    const stepIndex = this.stepper()?.selectedIndex ?? 0;
    const adjustedIndex = this.showCustomerStep() ? stepIndex : stepIndex + 1;

    switch (adjustedIndex) {
      case 0:
        // Step 1: Customer Selection
        this.continueFromCustomerStep();
        break;
      case 1:
        // Step 2: Application Selection
        this.continueFromApplicationStep();
        break;
      case 2:
        // Step 3: Deployment Target Configuration
        this.continueFromDeploymentTargetStep();
        break;
      case 3:
        // Step 4: Application Configuration
        await this.continueFromApplicationConfigStep();
        break;
      case 4:
        // Step 5: Connect and Deploy
        await this.continueFromConnectStep();
        break;
    }
  }

  attemptGoBack() {
    if (this.loading) {
      return;
    }
    this.stepper()?.previous();
  }

  private continueFromCustomerStep() {
    this.customerForm.markAllAsTouched();
    if (!this.customerForm.value.customerOrganizationId && !this.customerForm.value.internal) {
      return;
    }
    this.nextStep();
  }

  private continueFromApplicationStep() {
    this.applicationForm.markAllAsTouched();
    if (!this.applicationForm.valid) {
      return;
    }
    this.nextStep();
  }

  private continueFromDeploymentTargetStep() {
    this.deploymentTargetForm.markAllAsTouched();
    if (!this.deploymentTargetForm.valid) {
      return;
    }
    this.applications.refresh();
    this.nextStep();
  }

  private async continueFromApplicationConfigStep() {
    this.applicationConfigForm.markAllAsTouched();
    if (!this.applicationConfigForm.valid || this.loading) {
      return;
    }

    this.loading = true;
    let createdDeploymentTarget: DeploymentTarget | null = null;

    try {
      const app = this.selectedApplication();
      if (!app) {
        throw new Error('No application selected');
      }

      const customerOrgId = this.selectedCustomerOrganizationId();

      // Create deployment target
      try {
        createdDeploymentTarget = (await firstValueFrom(
          this.deploymentTargets.create({
            name: this.deploymentTargetForm.value.name!,
            type: app.type,
            namespace: this.deploymentTargetForm.value.namespace || undefined,
            scope: this.deploymentTargetForm.value.scope,
            deployments: [],
            metricsEnabled: this.deploymentTargetForm.value.scope !== 'namespace',
            imageCleanupEnabled: app.type === 'docker' && this.deploymentTargetForm.controls.imageCleanupEnabled.value,
            autohealEnabled: app.type === 'docker' ? (this.deploymentTargetForm.value.autohealEnabled ?? true) : false,
            customerOrganization: customerOrgId ? ({id: customerOrgId} as CustomerOrganization) : undefined,
            resources: this.deploymentTargetForm.value.customResources
              ? {
                  cpuLimit: this.deploymentTargetForm.value.resources?.cpuLimit!,
                  memoryLimit: this.deploymentTargetForm.value.resources?.memoryLimit!,
                  cpuRequest: this.deploymentTargetForm.value.resources?.cpuRequest!,
                  memoryRequest: this.deploymentTargetForm.value.resources?.memoryRequest!,
                }
              : undefined,
          })
        )) as DeploymentTarget;
      } catch (e) {
        const msg = getFormDisplayedError(e);
        this.toast.error(msg || 'Failed to create deployment target');
        return;
      }

      // Deploy the application
      const deploymentFormData = this.applicationConfigForm.value.deploymentFormData;

      if (!deploymentFormData) {
        throw new Error('Missing deployment configuration');
      }

      const deployment = mapToDeploymentRequest(deploymentFormData, createdDeploymentTarget.id!);

      try {
        await firstValueFrom(this.deploymentTargets.deploy(deployment));
        this.selectedDeploymentTarget.set(createdDeploymentTarget);
        this.toast.success('Deployment created successfully');
        this.nextStep();
      } catch (e) {
        // Delete the deployment target if deployment fails
        const deployErrorMsg = getFormDisplayedError(e);
        this.toast.error(deployErrorMsg || 'Failed to deploy application');
        try {
          await firstValueFrom(this.deploymentTargets.delete(createdDeploymentTarget));
          this.selectedDeploymentTarget.set(undefined);
        } catch (deleteError) {
          const msg = getFormDisplayedError(deleteError);
          this.toast.error(
            `The following error occurred trying to clean up a failed deployment: '${msg}'. Please close this dialog and clean up the deployment target manually.`
          );
        }
      }
    } finally {
      this.loading = false;
    }
  }

  private async continueFromConnectStep() {
    this.close();
  }

  close() {
    this.closed.emit();
  }

  private nextStep() {
    this.loading = false;
    this.stepper()?.next();
  }

  selectApplication(app: Application) {
    this.applicationForm.controls.applicationId.setValue(app.id!);
  }

  selectCustomer(customer: CustomerOrganization | null) {
    this.customerForm.controls.customerOrganizationId.setValue(customer?.id);
    this.customerForm.controls.internal.setValue(!customer?.id);
  }
}
