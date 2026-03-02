import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {Directive, inject, input, signal, TemplateRef, viewChild} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, Validators} from '@angular/forms';
import {
  DeploymentTarget,
  DeploymentTargetScope,
  DeploymentType,
  DeploymentWithLatestRevision,
} from '@distr-sh/distr-sdk';
import {
  faArrowUpRightFromSquare,
  faCircleExclamation,
  faComment,
  faLink,
  faPen,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, lastValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {RESOURCE_QUANTITY_REGEX} from '../../../util/validation';
import {AuthService} from '../../services/auth.service';
import {DeploymentTargetLatestMetrics} from '../../services/deployment-target-metrics.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';

@Directive()
export abstract class DeploymentTargetCardBaseComponent {
  protected readonly overlay = inject(OverlayService);
  protected readonly auth = inject(AuthService);
  protected readonly deploymentTargets = inject(DeploymentTargetsService);
  protected readonly toast = inject(ToastService);

  protected readonly customerManagedWarning = `
    You are about to make changes to a customer-managed deployment.
    Ensure this is done in coordination with the customer.`;

  public readonly deploymentTarget = input.required<DeploymentTarget>();
  public readonly deploymentTargetMetrics = input<DeploymentTargetLatestMetrics | undefined>(undefined);

  protected readonly deploymentModal = viewChild.required<TemplateRef<unknown>>('deploymentModal');
  protected readonly deploymentStatusModal = viewChild.required<TemplateRef<unknown>>('deploymentStatusModal');
  protected readonly deploymentTargetStatusModal =
    viewChild.required<TemplateRef<unknown>>('deploymentTargetStatusModal');
  protected readonly instructionsModal = viewChild.required<TemplateRef<unknown>>('instructionsModal');
  protected readonly deleteConfirmModal = viewChild.required<TemplateRef<unknown>>('deleteConfirmModal');
  protected readonly manageDeploymentTargetDrawer =
    viewChild.required<TemplateRef<unknown>>('manageDeploymentTargetDrawer');
  protected readonly deploymentTargetNotesDrawer =
    viewChild.required<TemplateRef<unknown>>('deploymentTargetNotesDrawer');

  protected readonly faArrowUpRightFromSquare = faArrowUpRightFromSquare;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faComment = faComment;
  protected readonly faLink = faLink;
  protected readonly faPen = faPen;
  protected readonly faXmark = faXmark;

  protected readonly selectedDeploymentTarget = signal<DeploymentTarget | undefined>(undefined);
  protected readonly selectedDeployment = signal<DeploymentWithLatestRevision | undefined>(undefined);

  protected readonly editForm = new FormGroup({
    id: new FormControl<string | undefined>(undefined),
    name: new FormControl('', Validators.required),
    type: new FormControl<DeploymentType | undefined>({value: undefined, disabled: true}, Validators.required),
    namespace: new FormControl<string | undefined>({value: undefined, disabled: true}),
    scope: new FormControl<DeploymentTargetScope>({value: 'namespace', disabled: true}),
    metricsEnabled: new FormControl<boolean>(true),
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

  private loadDeploymentTarget(dt: DeploymentTarget) {
    this.editForm.patchValue({
      ...dt,
      customResources: !!dt.resources,
    });
    if (dt.scope === 'namespace') {
      this.editForm.controls.metricsEnabled.disable();
    } else {
      this.editForm.controls.metricsEnabled.enable();
    }
    if (dt.type === 'kubernetes') {
      this.editForm.controls.customResources.enable();
    } else {
      this.editForm.controls.customResources.setValue(false);
      this.editForm.controls.customResources.disable();
    }
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

  private resetEditForm() {
    this.editForm.reset();
    this.editForm.patchValue({type: 'docker'});
  }
}
