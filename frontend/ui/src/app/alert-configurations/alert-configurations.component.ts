import { DatePipe } from '@angular/common';
import { Component, computed, inject, Signal, signal, TemplateRef, viewChild } from '@angular/core';
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, FormControl, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { CustomerOrganization, DeploymentTarget, Named } from '@distr-sh/distr-sdk';
import { FaIconComponent } from '@fortawesome/angular-fontawesome';
import {
  faCheck,
  faClockRotateLeft,
  faMagnifyingGlass,
  faPen,
  faPlus,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import { firstValueFrom, startWith, Subject, switchMap } from 'rxjs';
import { getFormDisplayedError } from '../../util/errors';
import { validateRecordAtLeast } from '../../util/validation';
import { AlertConfigurationsService } from '../services/alert-configurations.service';
import { AuthService } from '../services/auth.service';
import { CustomerOrganizationsService } from '../services/customer-organizations.service';
import { DeploymentTargetsService } from '../services/deployment-targets.service';
import { DialogRef, OverlayService } from '../services/overlay.service';
import { ToastService } from '../services/toast.service';
import { UsersService } from '../services/users.service';
import { AlertConfiguration, CreateUpdateAlertConfigurationRequest } from '../types/alert-configuration';

@Component({
  templateUrl: './alert-configurations.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, DatePipe, RouterLink],
})
export class AlertConfigurationsComponent {
  protected readonly auth = inject(AuthService);
  private readonly svc = inject(AlertConfigurationsService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly overlay = inject(OverlayService);
  private readonly usersService = inject(UsersService);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly customersService = inject(CustomerOrganizationsService);
  private readonly toast = inject(ToastService);

  protected readonly editConfigRef = signal<AlertConfiguration | undefined>(undefined);
  protected readonly editConfigForm = this.fb.group(
    {
      id: this.fb.control(''),
      name: this.fb.control(''),
      enabled: this.fb.control(true),
      statusTriggerEnabled: this.fb.control(true),
      cpuTriggerEnabled: this.fb.control(false),
      cpuTriggerThresholdPercent: this.fb.control(75, [Validators.min(1), Validators.max(100), Validators.required]),
      memoryTriggerEnabled: this.fb.control(false),
      memoryTriggerThresholdPercent: this.fb.control(75, [Validators.min(1), Validators.max(100), Validators.required]),
      diskTriggerEnabled: this.fb.control(false),
      diskTriggerThresholdPercent: this.fb.control(75, [Validators.min(1), Validators.max(100), Validators.required]),
      userAccountIds: this.fb.record<boolean>({}, { validators: [validateRecordAtLeast(1)] }),
      deploymentTargetIds: this.fb.record<boolean>({}, { validators: [validateRecordAtLeast(1)] }),
    },
    {
      validators: [
        (ctrl) => {
          const v = ctrl.value;
          return v.statusTriggerEnabled || v.cpuTriggerEnabled || v.memoryTriggerEnabled || v.diskTriggerEnabled
            ? null
            : { anyTrigger: 1 };
        },
      ],
    }
  );

  constructor() {
    const thresholdPairs: [FormControl<boolean>, FormControl<number | null>][] = [
      [this.editConfigForm.controls.cpuTriggerEnabled, this.editConfigForm.controls.cpuTriggerThresholdPercent],
      [this.editConfigForm.controls.memoryTriggerEnabled, this.editConfigForm.controls.memoryTriggerThresholdPercent],
      [this.editConfigForm.controls.diskTriggerEnabled, this.editConfigForm.controls.diskTriggerThresholdPercent],
    ];
    for (const [enabledCtrl, thresholdCtrl] of thresholdPairs) {
      enabledCtrl.valueChanges.pipe(takeUntilDestroyed()).subscribe((enabled) => {
        if (enabled) {
          thresholdCtrl.enable();
          if (!thresholdCtrl.value) {
            thresholdCtrl.setValue(75);
          }
        } else {
          thresholdCtrl.disable();
        }
      });
    }
  }
  protected readonly editFormLoading = signal(false);
  private readonly editConfigDrawerTpl = viewChild.required<TemplateRef<unknown>>('editConfigDrawer');
  private editConfigDrawerRef?: DialogRef;

  protected readonly enabledToggleLoading = signal(false);

  private readonly reload$ = new Subject<void>();
  protected readonly configs = toSignal(
    this.reload$.pipe(
      startWith(undefined),
      switchMap(() => this.svc.list())
    )
  );
  private readonly allUsers = toSignal(this.usersService.getUsers());
  protected readonly users = this.auth.isCustomer()
    ? this.allUsers
    : computed(() => this.allUsers()?.filter((it) => it.customerOrganizationId === undefined));
  private readonly deploymentTargets = toSignal(this.deploymentTargetsService.list());
  private readonly customers = this.auth.isVendor()
    ? toSignal(this.customersService.getCustomerOrganizations())
    : signal([]).asReadonly();
  protected readonly deploymentTargetCustomers: Signal<
    { customer?: CustomerOrganization; deploymentTargets: DeploymentTarget[] }[]
  > = computed(() => {
    const deploymentTargets = this.deploymentTargets() ?? [];
    const customers = this.customers();
    return customers?.length
      ? [
        { deploymentTargets: deploymentTargets.filter((it) => it.customerOrganization === undefined) },
        ...customers.map((customer) => ({
          customer,
          deploymentTargets: deploymentTargets.filter((it) => it.customerOrganization?.id === customer.id),
        })),
      ].filter((entry) => entry.deploymentTargets.length > 0)
      : [{ deploymentTargets }];
  });

  protected readonly filterForm = this.fb.group({
    search: '',
  });

  private readonly filterValue = toSignal(this.filterForm.controls.search.valueChanges);

  protected readonly filteredConfigs = computed(() => {
    const value = this.filterValue()?.toLowerCase();
    const configs = this.configs();
    return !value ? configs : configs?.filter((it) => it.name.toLowerCase().includes(value));
  });

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPlus = faPlus;
  protected readonly faPen = faPen;
  protected readonly faTrash = faTrash;
  protected readonly faCheck = faCheck;
  protected readonly faXmark = faXmark;
  protected readonly faHistory = faClockRotateLeft;

  protected async showDrawer(config?: AlertConfiguration) {
    this.hideDrawer();
    this.editConfigRef.set(config);
    this.editConfigForm.reset();

    this.users()
      ?.filter((user) => user.customerOrganizationId === this.auth.getClaims()?.c_org)
      .map((user) => user.id)
      .filter((id) => id !== undefined)
      .forEach((id) => this.editConfigForm.controls.userAccountIds.addControl(id, this.fb.control(false)));

    this.deploymentTargets()
      ?.map((dt) => dt.id)
      .filter((id) => id !== undefined)
      .forEach((id) => this.editConfigForm.controls.deploymentTargetIds.addControl(id, this.fb.control(false)));

    console.log(config);
    if (config) {
      this.editConfigForm.patchValue({
        id: config.id,
        name: config.name,
        enabled: config.enabled,
        statusTriggerEnabled: config.statusTriggerEnabled,
        cpuTriggerEnabled: config.cpuTriggerThresholdPercent !== undefined,
        cpuTriggerThresholdPercent: config.cpuTriggerThresholdPercent,
        memoryTriggerEnabled: config.memoryTriggerThresholdPercent !== undefined,
        memoryTriggerThresholdPercent: config.memoryTriggerThresholdPercent,
        diskTriggerEnabled: config.diskTriggerThresholdPercent !== undefined,
        diskTriggerThresholdPercent: config.diskTriggerThresholdPercent,
        deploymentTargetIds: config.deploymentTargetIds?.reduce((acc, id) => ({ ...acc, [id]: true }), {}),
        userAccountIds: config.userAccountIds?.reduce((acc, id) => ({ ...acc, [id]: true }), {}),
      });
    }

    this.editConfigDrawerRef = this.overlay.showDrawer(this.editConfigDrawerTpl());
  }

  protected hideDrawer() {
    this.editConfigDrawerRef?.dismiss();
  }

  protected async saveConfig() {
    if (this.editConfigForm.invalid) {
      this.editConfigForm.markAllAsTouched();
      return;
    }

    const formValue = this.editConfigForm.value;
    const requestValue: CreateUpdateAlertConfigurationRequest = {
      name: formValue.name ?? '',
      enabled: formValue.enabled ?? true,
      statusTriggerEnabled: formValue.statusTriggerEnabled ?? false,
      cpuTriggerThresholdPercent: formValue.cpuTriggerThresholdPercent,
      memoryTriggerThresholdPercent: formValue.memoryTriggerThresholdPercent,
      diskTriggerThresholdPercent: formValue.diskTriggerThresholdPercent,
      deploymentTargetIds: Object.entries(formValue.deploymentTargetIds ?? {})
        .filter(([_, checked]) => checked)
        .map(([id]) => id),
      userAccountIds: Object.entries(formValue.userAccountIds ?? {})
        .filter(([_, checked]) => checked)
        .map(([id]) => id),
    };

    this.editFormLoading.set(true);

    try {
      if (formValue.id) {
        await firstValueFrom(this.svc.update(formValue.id, requestValue));
        this.toast.success('Alert configuration updated');
      } else {
        await firstValueFrom(this.svc.create(requestValue));
        this.toast.success('Alert configuration created');
      }
      this.reload$.next();
      this.hideDrawer();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.editFormLoading.set(false);
    }
  }

  protected async toggleConfigEnabled(config: AlertConfiguration) {
    try {
      const request = { ...config, enabled: !config.enabled };
      this.enabledToggleLoading.set(true);
      await firstValueFrom(this.svc.update(config.id, request));
      this.toast.success(`Alert configuration ${request.enabled ? 'enabled' : 'disabled'}`);
      this.reload$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.enabledToggleLoading.set(false);
    }
  }

  protected async deleteConfig(config: AlertConfiguration) {
    this.svc.delete(config.id).subscribe({
      next: () => {
        this.toast.success('Alert configuration deleted');
        this.reload$.next();
      },
      error: (e) => this.toast.error(e),
    });
  }

  protected getTriggersCount(config: AlertConfiguration): number {
    return [
      config.statusTriggerEnabled,
      config.cpuTriggerThresholdPercent !== undefined,
      config.memoryTriggerThresholdPercent !== undefined,
      config.diskTriggerThresholdPercent !== undefined,
    ].filter(v => v).length
  }

  protected getNames(named: Named[] | undefined) {
    return named?.map((it) => it.name).join('\n');
  }
}
