import {DatePipe} from '@angular/common';
import {Component, computed, inject, Signal, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {CustomerOrganization, DeploymentTarget, Named} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faCheck,
  faClockRotateLeft,
  faMagnifyingGlass,
  faPen,
  faPlus,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith, Subject, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {validateRecordAtLeast} from '../../util/validation';
import {AuthService} from '../services/auth.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {DeploymentStatusNotificationConfigurationsService} from '../services/deployment-status-notification-configurations.service';
import {DeploymentTargetsService} from '../services/deployment-targets.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {UsersService} from '../services/users.service';
import {
  CreateUpdateDeploymentStatusNotificationConfigurationRequest,
  DeploymentStatusNotificationConfiguration,
} from '../types/deployment-status-notification-configuration';

@Component({
  templateUrl: './deployment-status-notification-configurations.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, DatePipe, RouterLink],
})
export class DeploymentStatusNotificationConfigurationsComponent {
  protected readonly auth = inject(AuthService);
  private readonly svc = inject(DeploymentStatusNotificationConfigurationsService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly overlay = inject(OverlayService);
  private readonly usersService = inject(UsersService);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly customersService = inject(CustomerOrganizationsService);
  private readonly toast = inject(ToastService);

  protected readonly editConfigRef = signal<DeploymentStatusNotificationConfiguration | undefined>(undefined);
  protected readonly editConfigForm = this.fb.group({
    id: this.fb.control(''),
    name: this.fb.control(''),
    enabled: this.fb.control(true),
    userAccountIds: this.fb.record<boolean>({}, {validators: [validateRecordAtLeast(1)]}),
    deploymentTargetIds: this.fb.record<boolean>({}, {validators: [validateRecordAtLeast(1)]}),
  });
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
    {customer?: CustomerOrganization; deploymentTargets: DeploymentTarget[]}[]
  > = computed(() => {
    const deploymentTargets = this.deploymentTargets() ?? [];
    const customers = this.customers();
    return customers?.length
      ? [
          {deploymentTargets: deploymentTargets.filter((it) => it.customerOrganization === undefined)},
          ...customers.map((customer) => ({
            customer,
            deploymentTargets: deploymentTargets.filter((it) => it.customerOrganization?.id === customer.id),
          })),
        ].filter((entry) => entry.deploymentTargets.length > 0)
      : [{deploymentTargets}];
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

  protected async showDrawer(config?: DeploymentStatusNotificationConfiguration) {
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

    if (config) {
      this.editConfigForm.patchValue({
        id: config.id,
        name: config.name,
        enabled: config.enabled,
        deploymentTargetIds: config.deploymentTargetIds?.reduce((acc, id) => ({...acc, [id]: true}), {}),
        userAccountIds: config.userAccountIds?.reduce((acc, id) => ({...acc, [id]: true}), {}),
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
    const requestValue: CreateUpdateDeploymentStatusNotificationConfigurationRequest = {
      name: formValue.name ?? '',
      enabled: formValue.enabled ?? true,
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
        this.toast.success('Deployment status notification configuration updated');
      } else {
        await firstValueFrom(this.svc.create(requestValue));
        this.toast.success('Deployment status notification configuration created');
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

  protected async toggleConfigEnabled(config: DeploymentStatusNotificationConfiguration) {
    console.log('Toggling enabled for config', config);
    try {
      const request = {...config, enabled: !config.enabled};
      this.enabledToggleLoading.set(true);
      await firstValueFrom(this.svc.update(config.id, request));
      this.toast.success(`Deployment status notification configuration ${request.enabled ? 'enabled' : 'disabled'}`);
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

  protected async deleteConfig(config: DeploymentStatusNotificationConfiguration) {
    this.svc.delete(config.id).subscribe({
      next: () => {
        this.toast.success('Deployment status notification configuration deleted');
        this.reload$.next();
      },
      error: (e) => this.toast.error(e),
    });
  }

  protected getNames(named: Named[] | undefined) {
    return named?.map((it) => it.name).join('\n');
  }
}
