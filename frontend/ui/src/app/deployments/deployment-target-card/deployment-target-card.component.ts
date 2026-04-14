import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {TextFieldModule} from '@angular/cdk/text-field';
import {DatePipe, NgOptimizedImage} from '@angular/common';
import {
  Component,
  computed,
  inject,
  input,
  resource,
  signal,
  TemplateRef,
  viewChild,
  WritableSignal,
} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {
  ApplicationVersion,
  DeploymentTarget,
  DeploymentTargetScope,
  DeploymentType,
  DeploymentWithLatestRevision,
} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArrowUpRightFromSquare,
  faBinoculars,
  faCircleExclamation,
  faComment,
  faEllipsisVertical,
  faGear,
  faHeartPulse,
  faLink,
  faPen,
  faPlus,
  faRotate,
  faTrash,
  faTriangleExclamation,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {EMPTY, filter, firstValueFrom, lastValueFrom, switchMap} from 'rxjs';
import {SemVer} from 'semver';
import {maxBy} from '../../../util/arrays';
import {dateTimeLocalToISO, isArchived, isoToDateTimeLocal} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {RESOURCE_QUANTITY_REGEX} from '../../../util/validation';
import {ConnectInstructionsComponent} from '../../components/connect-instructions/connect-instructions.component';
import {SpinnerComponent} from '../../components/spinner/spinner.component';
import {DeploymentTargetStatusDotComponent} from '../../components/status-dot';
import {UuidComponent} from '../../components/uuid';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AgentVersionService} from '../../services/agent-version.service';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ApplicationsService} from '../../services/applications.service';
import {AuthService} from '../../services/auth.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {DeploymentTargetLatestMetrics} from '../../types/deployment-target-metrics';
import {DeploymentModalComponent} from '../deployment-modal.component';
import {DeploymentAppNameComponent} from './deployment-app-name.component';
import {DeploymentStatusTextComponent} from './deployment-status-text.component';
import {DeploymentTargetMetricsComponent} from './deployment-target-metrics.component';

@Component({
  selector: 'app-deployment-target-card',
  templateUrl: './deployment-target-card.component.html',
  imports: [
    NgOptimizedImage,
    DeploymentTargetStatusDotComponent,
    UuidComponent,
    DatePipe,
    FaIconComponent,
    OverlayModule,
    ConnectInstructionsComponent,
    ReactiveFormsModule,
    DeploymentModalComponent,
    DeploymentTargetMetricsComponent,
    TextFieldModule,
    DeploymentAppNameComponent,
    DeploymentStatusTextComponent,
    AutotrimDirective,
    RouterLink,
    SpinnerComponent,
  ],
})
export class DeploymentTargetCardComponent {
  public readonly deploymentTarget = input.required<DeploymentTarget>();
  public readonly deploymentTargetMetrics = input<DeploymentTargetLatestMetrics | undefined>(undefined);

  protected readonly overlay = inject(OverlayService);
  protected readonly auth = inject(AuthService);
  private readonly deploymentTargets = inject(DeploymentTargetsService);
  private readonly toast = inject(ToastService);
  private readonly agentVersionsSvc = inject(AgentVersionService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly featureFlags = inject(FeatureFlagService);

  protected readonly customerManagedWarning = `
    You are about to make changes to a customer-managed deployment.
    Ensure this is done in coordination with the customer.`;

  protected readonly deploymentModal = viewChild.required<TemplateRef<unknown>>('deploymentModal');
  protected readonly instructionsModal = viewChild.required<TemplateRef<unknown>>('instructionsModal');
  protected readonly deleteConfirmModal = viewChild.required<TemplateRef<unknown>>('deleteConfirmModal');
  protected readonly manageDeploymentTargetDrawer =
    viewChild.required<TemplateRef<unknown>>('manageDeploymentTargetDrawer');
  protected readonly deploymentTargetNotesDrawer =
    viewChild.required<TemplateRef<unknown>>('deploymentTargetNotesDrawer');
  protected readonly deleteDeploymentProgressModal = viewChild.required<TemplateRef<unknown>>(
    'deleteDeploymentProgressModal'
  );

  protected readonly faArrowUpRightFromSquare = faArrowUpRightFromSquare;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faComment = faComment;
  protected readonly faBinoculars = faBinoculars;
  protected readonly faEllipsisVertical = faEllipsisVertical;
  protected readonly faGear = faGear;
  protected readonly faHeartPulse = faHeartPulse;
  protected readonly faLink = faLink;
  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faRotate = faRotate;
  protected readonly faTrash = faTrash;
  protected readonly faTriangleExclamation = faTriangleExclamation;
  protected readonly faXmark = faXmark;

  protected readonly selectedDeploymentTarget = signal<DeploymentTarget | undefined>(undefined);
  protected readonly selectedDeployment = signal<DeploymentWithLatestRevision | undefined>(undefined);

  protected readonly showDeploymentTargetDropdown = signal(false);
  protected readonly showDeploymentDropdownForId = signal<string | undefined>(undefined);
  protected readonly metricsOpened = signal(false);

  protected readonly editForm = new FormGroup({
    id: new FormControl<string | undefined>(undefined),
    name: new FormControl('', Validators.required),
    type: new FormControl<DeploymentType | undefined>({value: undefined, disabled: true}, Validators.required),
    namespace: new FormControl<string | undefined>({value: undefined, disabled: true}),
    scope: new FormControl<DeploymentTargetScope>({value: 'namespace', disabled: true}),
    metricsEnabled: new FormControl<boolean>(true),
    imageCleanupEnabled: new FormControl<boolean>(false, {nonNullable: true}),
    deploymentLogsEnabled: new FormControl<boolean>(true, {nonNullable: true}),
    deploymentLogsAfter: new FormControl<string | null>(null),
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
  });
  protected editFormLoading = false;

  protected readonly notesForm = new FormGroup({
    notes: new FormControl<string>(
      {value: '', disabled: !this.auth.hasAnyRole('admin', 'read_write')},
      {nonNullable: true}
    ),
  });
  protected notesFormLoading = false;

  private modal?: DialogRef;
  private drawerRef?: DialogRef;

  protected readonly agentVersions = resource({
    loader: () => firstValueFrom(this.agentVersionsSvc.list()),
  });

  protected readonly isUndeploySupported = this.isAgentVersionAtLeast('1.3.0');
  protected readonly isMultiDeploymentSupported = this.isAgentVersionAtLeast('1.6.0');
  protected readonly isLoggingSupported = this.isAgentVersionAtLeast('1.9.0');
  protected readonly isForceRestartSupported = this.isAgentVersionAtLeast('1.12.0');

  protected readonly agentUpdateAvailable = computed(() => {
    const agentVersions = this.agentVersions.value() ?? [];
    return (
      agentVersions.length > 0 &&
      this.deploymentTarget().agentVersion?.id !== agentVersions[agentVersions.length - 1].id
    );
  });

  private readonly entitlements = toSignal(
    this.featureFlags.isLicensingEnabled$.pipe(
      switchMap((enabled) => (enabled ? this.applicationEntitlementsService.list() : EMPTY))
    ),
    {initialValue: []}
  );

  private readonly applications = toSignal(this.applicationsService.list(), {initialValue: []});

  protected readonly deploymentIdsWithUpdate = computed(() => {
    const deploymentTarget = this.deploymentTarget();
    const applications = this.applications();
    const entitlements = this.entitlements();

    return new Set(
      deploymentTarget.deployments
        .map((deployment) => {
          const applicationVersions =
            (deployment.applicationEntitlementId
              ? entitlements.find((entitlement) => entitlement.id === deployment.applicationEntitlementId)?.application
                  ?.versions
              : undefined) ?? applications.find((app) => app.id === deployment.application.id)?.versions;

          const maxVersion = this.findMaxVersion(applicationVersions?.filter((version) => !isArchived(version)) ?? []);

          return maxVersion && deployment.applicationVersionId !== maxVersion.id ? deployment.id : undefined;
        })
        .filter((id) => id !== undefined)
    );
  });

  protected readonly agentUpdatePending = computed(
    () =>
      this.deploymentTarget().currentStatus !== undefined &&
      this.deploymentTarget().agentVersion?.id !== this.deploymentTarget().reportedAgentVersionId
  );

  constructor() {
    this.editForm.controls.customResources.valueChanges.pipe(takeUntilDestroyed()).subscribe((value) => {
      if (value) {
        this.editForm.controls.resources.enable();
      } else {
        this.editForm.controls.resources.disable();
      }
    });
  }

  protected async showDeploymentModal(deployment?: DeploymentWithLatestRevision) {
    this.selectedDeploymentTarget.set(this.deploymentTarget());
    this.selectedDeployment.set(deployment);
    this.showModal(this.deploymentModal());
  }

  protected async saveDeploymentTarget() {
    this.editForm.markAllAsTouched();
    if (this.editForm.valid) {
      this.editFormLoading = true;
      const val = this.editForm.value;
      const dt: DeploymentTarget = {
        id: val.id!,
        name: val.name!,
        type: val.type!,
        deployments: [],
        metricsEnabled: val.metricsEnabled ?? false,
        imageCleanupEnabled: this.editForm.controls.imageCleanupEnabled.value,
        deploymentLogsEnabled: this.editForm.controls.deploymentLogsEnabled.value,
        deploymentLogsAfter: dateTimeLocalToISO(this.editForm.controls.deploymentLogsAfter.value) ?? undefined,
        resources: val.resources && {
          cpuRequest: val.resources.cpuRequest!,
          cpuLimit: val.resources.cpuLimit!,
          memoryRequest: val.resources.memoryRequest!,
          memoryLimit: val.resources.memoryLimit!,
        },
      };

      try {
        this.loadDeploymentTarget(
          await lastValueFrom(val.id ? this.deploymentTargets.update(dt) : this.deploymentTargets.create(dt))
        );
        this.toast.success(`${dt.name} saved successfully`);
        this.hideDrawer();
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.editFormLoading = false;
      }
    }
  }

  protected saveDeploymentTargetNotes() {
    const id = this.deploymentTarget().id;
    const notes = this.notesForm.value.notes ?? '';

    if (!id) {
      return;
    }

    this.notesFormLoading = true;
    this.deploymentTargets.saveNotes(id, notes).subscribe({
      next: () => {
        this.toast.success('Notes saved successfully');
        this.notesFormLoading = false;
      },
      error: () => {
        this.toast.error('Failed to save notes');
        this.notesFormLoading = false;
      },
    });
  }

  protected showLogsEnabledMovedAlert() {
    this.overlay
      .confirm({
        message: {
          message: 'Log collection is now configured per agent. Open the agent settings to enable or disable it.',
        },
        confirmLabel: 'Open Settings',
        cancelLabel: 'Close',
      })
      .pipe(filter((result) => result === true))
      .subscribe(() => this.openEditDrawer());
  }

  protected forceRestart(deployment: DeploymentWithLatestRevision) {
    if (deployment.id) {
      this.overlay
        .confirm({
          message: {
            message: 'Are you sure you want to force restart this deployment?',
            alert: {
              type: 'warning',
              message: 'Depending on the deployment, this may cause downtime.',
            },
          },
        })
        .pipe(
          filter((result) => result === true),
          switchMap(() =>
            this.deploymentTargets.deploy({
              deploymentId: deployment.id,
              deploymentTargetId: deployment.deploymentTargetId,
              applicationVersionId: deployment.applicationVersionId,
              applicationEntitlementId: deployment.applicationEntitlementId,
              releaseName: deployment.releaseName,
              dockerType: deployment.dockerType,
              valuesYaml: deployment.valuesYaml,
              envFileData: deployment.envFileData,
              forceRestart: true,
            })
          )
        )
        .subscribe({
          next: () => this.toast.success('Deployment has been restarted.'),
          error: (e) => {
            const msg = getFormDisplayedError(e);
            if (msg) {
              this.toast.error(msg);
            }
          },
        });
    }
  }

  protected deleteDeploymentTarget() {
    const dt = this.deploymentTarget();
    const alert =
      dt.customerOrganization !== undefined && this.auth.isVendor()
        ? ({type: 'warning', message: this.customerManagedWarning} as const)
        : undefined;
    this.overlay
      .confirm({
        customTemplate: this.deleteConfirmModal(),
        requiredConfirmInputText: 'DELETE',
        message: {
          alert,
          message: '',
        },
      })
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.deploymentTargets.delete(dt))
      )
      .subscribe({
        error: (e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
  }

  protected async deleteDeployment(d: DeploymentWithLatestRevision, confirmTemplate: TemplateRef<any>) {
    const dt = this.deploymentTarget();
    const alert =
      dt.customerOrganization !== undefined && this.auth.isVendor()
        ? ({type: 'warning', message: this.customerManagedWarning} as const)
        : undefined;
    if (d.id) {
      if (
        await firstValueFrom(
          this.overlay.confirm({
            customTemplate: confirmTemplate,
            requiredConfirmInputText: 'UNDEPLOY',
            message: {
              alert,
              message: '',
            },
          })
        )
      ) {
        const modalRef = this.overlay.showModal(this.deleteDeploymentProgressModal(), {
          positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
          backdropStyleOnly: true,
        });

        try {
          await firstValueFrom(this.deploymentTargets.undeploy(d.id));
        } catch (e) {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        } finally {
          modalRef?.dismiss();
        }
      }
    }
  }

  public async updateDeploymentTargetAgent(): Promise<void> {
    try {
      const dt = this.deploymentTarget();
      const agentVersions = this.agentVersions.value();
      if (agentVersions?.length) {
        const targetVersion = agentVersions[agentVersions.length - 1];
        if (
          await firstValueFrom(
            this.overlay.confirm(`Update ${dt.name} agent from ${dt.agentVersion?.name} to ${targetVersion.name}?`)
          )
        ) {
          dt.agentVersion = targetVersion;
          await firstValueFrom(this.deploymentTargets.update(dt));
        }
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected toggle(signal: WritableSignal<boolean>) {
    signal.update((val) => !val);
  }

  protected async openInstructionsModal() {
    const dt = this.deploymentTarget();
    if (dt.currentStatus !== undefined) {
      const message = `If you continue, the previous authentication secret for ${dt.name} becomes invalid. Continue?`;
      const alert =
        dt.customerOrganization !== undefined && this.auth.isVendor()
          ? ({type: 'warning', message: this.customerManagedWarning} as const)
          : undefined;
      if (!(await firstValueFrom(this.overlay.confirm({message: {message, alert}})))) {
        return;
      }
    }
    this.showModal(this.instructionsModal());
  }

  protected showModal(templateRef: TemplateRef<unknown>) {
    this.hideModal();
    this.modal = this.overlay.showModal(templateRef, {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  protected hideModal(): void {
    this.modal?.close();
  }

  protected openEditDrawer() {
    this.hideDrawer();
    this.loadDeploymentTarget(this.deploymentTarget());
    this.drawerRef = this.overlay.showDrawer(this.manageDeploymentTargetDrawer());
  }

  protected openNotesDrawer() {
    const id = this.deploymentTarget().id;
    if (!id) return;
    this.hideDrawer();
    this.drawerRef = this.overlay.showDrawer(this.deploymentTargetNotesDrawer());
    this.notesFormLoading = true;
    this.deploymentTargets.getNotes(id).subscribe({
      next: (notes) => {
        this.notesForm.patchValue(notes);
        this.notesFormLoading = false;
      },
      error: () => {
        this.toast.error('Failed to load notes');
        this.notesFormLoading = false;
      },
    });
  }

  protected hideDrawer() {
    this.drawerRef?.close();
    this.resetEditForm();
    this.notesForm.reset();
  }

  protected setDeploymentLogsAfterNow() {
    this.editForm.controls.deploymentLogsAfter.setValue(isoToDateTimeLocal(dayjs().toISOString()));
  }

  protected setDeploymentLogsAfterNowMinus1h() {
    this.editForm.controls.deploymentLogsAfter.setValue(isoToDateTimeLocal(dayjs().subtract(1, 'hour').toISOString()));
  }

  protected clearDeploymentLogsAfter() {
    this.editForm.controls.deploymentLogsAfter.setValue(null);
  }

  private loadDeploymentTarget(dt: DeploymentTarget) {
    this.editForm.patchValue({
      ...dt,
      deploymentLogsEnabled: dt.deploymentLogsEnabled,
      deploymentLogsAfter: isoToDateTimeLocal(dt.deploymentLogsAfter) || null,
      customResources: !!dt.resources,
    });
    if (dt.scope === 'namespace') {
      this.editForm.controls.metricsEnabled.disable();
    } else {
      this.editForm.controls.metricsEnabled.enable();
    }
    if (dt.type === 'kubernetes') {
      this.editForm.controls.imageCleanupEnabled.disable();
      this.editForm.controls.customResources.enable();
    } else {
      this.editForm.controls.imageCleanupEnabled.enable();
      this.editForm.controls.customResources.setValue(false);
      this.editForm.controls.customResources.disable();
    }
  }

  private resetEditForm() {
    this.editForm.reset();
    this.editForm.patchValue({type: 'docker'});
  }

  private isAgentVersionAtLeast(version: string, allowSnapshot = true) {
    return computed(() => {
      if (!this.deploymentTarget().reportedAgentVersionId) {
        console.warn('reported agent version id is empty');
        return true;
      }
      const reported = this.agentVersions
        .value()
        ?.find((it) => it.id === this.deploymentTarget().reportedAgentVersionId);
      if (!reported) {
        console.warn('agent version with id not found', this.deploymentTarget().reportedAgentVersionId);
        return false;
      }
      try {
        return (allowSnapshot && reported.name === 'snapshot') || new SemVer(reported.name).compare(version) >= 0;
      } catch (e) {
        console.warn(e);
        return allowSnapshot && reported.name === 'snapshot';
      }
    });
  }

  private findMaxVersion(versions: ApplicationVersion[]): ApplicationVersion | undefined {
    try {
      return maxBy(
        versions,
        (version) => new SemVer(version.name),
        (a, b) => a.compare(b) > 0
      );
    } catch (e) {
      console.warn('semver compare failed, falling back to creation date', e);
      return maxBy(versions, (version) => new Date(version.createdAt!));
    }
  }
}
