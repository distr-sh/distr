import {DatePipe} from '@angular/common';
import {Component, computed, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClockRotateLeft, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith, Subject, switchMap} from 'rxjs';
import {compareBy} from '../../util/arrays';
import {getFormDisplayedError} from '../../util/errors';
import {checkedIds, checkedRecord} from '../../util/formRecord';
import {validateRecordAtLeast} from '../../util/validation';
import {ApplicationPreviewComponent} from '../applications/components';
import {SearchBarComponent} from '../components/search-bar.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ApplicationsService} from '../services/applications.service';
import {AuthService} from '../services/auth.service';
import {ApplicationNotificationConfigurationsService} from '../services/notification-configurations.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {UsersService} from '../services/users.service';
import {
  ApplicationNotificationConfiguration,
  CreateUpdateApplicationNotificationConfigurationRequest,
} from '../types/notification-configuration';
import {NotificationRecipientsComponent} from './notification-recipients.component';

@Component({
  templateUrl: './application-notification-configurations.component.html',
  imports: [
    FaIconComponent,
    ReactiveFormsModule,
    DatePipe,
    RouterLink,
    SearchBarComponent,
    AutotrimDirective,
    ApplicationPreviewComponent,
    NotificationRecipientsComponent,
  ],
})
export class ApplicationNotificationConfigurationsComponent {
  protected readonly auth = inject(AuthService);
  private readonly svc = inject(ApplicationNotificationConfigurationsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly usersService = inject(UsersService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  protected readonly faPlus = faPlus;
  protected readonly faPen = faPen;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly faHistory = faClockRotateLeft;

  private readonly reload$ = new Subject<void>();
  private readonly configs = toSignal(
    this.reload$.pipe(
      startWith(undefined),
      switchMap(() => this.svc.list())
    )
  );
  private readonly unsortedApplications = toSignal(this.applicationsService.list(), {initialValue: []});
  protected readonly applications = computed(() =>
    [...this.unsortedApplications()].sort(compareBy((application) => application.name!))
  );
  protected readonly users = toSignal(this.usersService.getUsers(), {initialValue: []});

  protected readonly filterForm = this.fb.group({search: ''});
  private readonly filterValue = toSignal(this.filterForm.controls.search.valueChanges);
  protected readonly filteredConfigs = computed(() => {
    const value = this.filterValue()?.toLowerCase();
    const configs = this.configs();
    return !value ? configs : configs?.filter((it) => it.name.toLowerCase().includes(value));
  });

  protected readonly editConfigRef = signal<ApplicationNotificationConfiguration | undefined>(undefined);
  protected readonly editFormLoading = signal(false);
  protected readonly enabledToggleLoading = signal(false);

  protected readonly editConfigForm = this.fb.group(
    {
      id: this.fb.control(''),
      name: this.fb.control('', [Validators.required]),
      enabled: this.fb.control(true),
      updateAvailableTriggerEnabled: this.fb.control(true),
      applicationIds: this.fb.record<boolean>({}, {validators: [validateRecordAtLeast(1)]}),
      userAccountIds: this.fb.record<boolean>({}, {validators: [validateRecordAtLeast(1)]}),
    },
    {validators: [(ctrl) => (ctrl.value.updateAvailableTriggerEnabled ? null : {anyTrigger: 1})]}
  );

  private readonly editConfigDrawerTpl = viewChild.required<TemplateRef<unknown>>('editConfigDrawer');
  private editConfigDrawerRef?: DialogRef;

  protected async showDrawer(config?: ApplicationNotificationConfiguration) {
    this.hideDrawer();
    this.editConfigRef.set(config);
    this.editConfigForm.reset();

    await Promise.all([this.applicationsService.refresh(), this.usersService.refresh()]);

    for (const application of this.applications()) {
      this.editConfigForm.controls.applicationIds.addControl(application.id!, this.fb.control(false));
    }
    for (const user of this.users()) {
      this.editConfigForm.controls.userAccountIds.addControl(user.id!, this.fb.control(false));
    }

    if (config) {
      this.editConfigForm.patchValue({
        id: config.id,
        name: config.name,
        enabled: config.enabled,
        updateAvailableTriggerEnabled: config.updateAvailableTriggerEnabled,
        applicationIds: checkedRecord(config.applications.map((it) => it.id)),
        userAccountIds: checkedRecord(config.recipients.map((it) => it.id)),
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

    const formValue = this.editConfigForm.getRawValue();
    const request: CreateUpdateApplicationNotificationConfigurationRequest = {
      name: formValue.name,
      enabled: formValue.enabled,
      updateAvailableTriggerEnabled: formValue.updateAvailableTriggerEnabled,
      applicationIds: checkedIds(formValue.applicationIds),
      userAccountIds: checkedIds(formValue.userAccountIds),
    };

    this.editFormLoading.set(true);
    try {
      if (formValue.id) {
        await firstValueFrom(this.svc.update(formValue.id, request));
        this.toast.success('Notification configuration updated');
      } else {
        await firstValueFrom(this.svc.create(request));
        this.toast.success('Notification configuration created');
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

  protected async toggleConfigEnabled(config: ApplicationNotificationConfiguration) {
    this.enabledToggleLoading.set(true);
    try {
      const enabled = !config.enabled;
      await firstValueFrom(
        this.svc.update(config.id, {
          name: config.name,
          enabled,
          updateAvailableTriggerEnabled: config.updateAvailableTriggerEnabled,
          applicationIds: config.applications.map((it) => it.id),
          userAccountIds: config.recipients.map((it) => it.id),
        })
      );
      this.toast.success(`Notification configuration ${enabled ? 'enabled' : 'disabled'}`);
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

  protected async deleteConfig(config: ApplicationNotificationConfiguration) {
    if (!(await firstValueFrom(this.overlay.confirm(`Really delete the notification "${config.name}"?`)))) {
      return;
    }

    try {
      await firstValueFrom(this.svc.delete(config.id));
      this.toast.success('Notification configuration deleted');
      this.reload$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}
