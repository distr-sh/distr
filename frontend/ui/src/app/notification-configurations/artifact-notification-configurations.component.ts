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
import {ArtifactPreviewComponent} from '../artifacts/components';
import {SearchBarComponent} from '../components/search-bar.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ArtifactsService} from '../services/artifacts.service';
import {AuthService} from '../services/auth.service';
import {ArtifactNotificationConfigurationsService} from '../services/notification-configurations.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {UsersService} from '../services/users.service';
import {
  ArtifactNotificationConfiguration,
  CreateUpdateArtifactNotificationConfigurationRequest,
} from '../types/notification-configuration';
import {NotificationRecipientsComponent} from './notification-recipients.component';

@Component({
  templateUrl: './artifact-notification-configurations.component.html',
  imports: [
    FaIconComponent,
    ReactiveFormsModule,
    DatePipe,
    RouterLink,
    SearchBarComponent,
    AutotrimDirective,
    ArtifactPreviewComponent,
    NotificationRecipientsComponent,
  ],
})
export class ArtifactNotificationConfigurationsComponent {
  protected readonly auth = inject(AuthService);
  private readonly svc = inject(ArtifactNotificationConfigurationsService);
  private readonly artifactsService = inject(ArtifactsService);
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
  private readonly unsortedArtifacts = toSignal(this.artifactsService.list(), {initialValue: []});
  // Mirrored artifacts are left out: every version of one appears at once when the upstream sync
  // runs, so announcing them would mean a notification per upstream tag.
  protected readonly artifacts = computed(() =>
    this.unsortedArtifacts()
      .filter((artifact) => !artifact.upstreamUrl)
      .sort(compareBy((artifact) => artifact.name))
  );
  protected readonly users = toSignal(this.usersService.getUsers(), {initialValue: []});

  protected readonly filterForm = this.fb.group({search: ''});
  private readonly filterValue = toSignal(this.filterForm.controls.search.valueChanges);
  protected readonly filteredConfigs = computed(() => {
    const value = this.filterValue()?.toLowerCase();
    const configs = this.configs();
    return !value ? configs : configs?.filter((it) => it.name.toLowerCase().includes(value));
  });

  protected readonly editConfigRef = signal<ArtifactNotificationConfiguration | undefined>(undefined);
  protected readonly editFormLoading = signal(false);
  protected readonly enabledToggleLoading = signal(false);

  protected readonly editConfigForm = this.fb.group(
    {
      id: this.fb.control(''),
      name: this.fb.control('', [Validators.required]),
      enabled: this.fb.control(true),
      newVersionTriggerEnabled: this.fb.control(true),
      artifactIds: this.fb.record<boolean>({}, {validators: [validateRecordAtLeast(1)]}),
      userAccountIds: this.fb.record<boolean>({}, {validators: [validateRecordAtLeast(1)]}),
    },
    {validators: [(ctrl) => (ctrl.value.newVersionTriggerEnabled ? null : {anyTrigger: 1})]}
  );

  private readonly editConfigDrawerTpl = viewChild.required<TemplateRef<unknown>>('editConfigDrawer');
  private editConfigDrawerRef?: DialogRef;

  protected async showDrawer(config?: ArtifactNotificationConfiguration) {
    this.hideDrawer();
    this.editConfigRef.set(config);
    this.editConfigForm.reset();

    await Promise.all([this.artifactsService.refresh(), this.usersService.refresh()]);

    for (const artifact of this.artifacts()) {
      this.editConfigForm.controls.artifactIds.addControl(artifact.id, this.fb.control(false));
    }
    for (const user of this.users()) {
      this.editConfigForm.controls.userAccountIds.addControl(user.id!, this.fb.control(false));
    }

    if (config) {
      this.editConfigForm.patchValue({
        id: config.id,
        name: config.name,
        enabled: config.enabled,
        newVersionTriggerEnabled: config.newVersionTriggerEnabled,
        artifactIds: checkedRecord(config.artifacts.map((it) => it.id)),
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
    const request: CreateUpdateArtifactNotificationConfigurationRequest = {
      name: formValue.name,
      enabled: formValue.enabled,
      newVersionTriggerEnabled: formValue.newVersionTriggerEnabled,
      artifactIds: checkedIds(formValue.artifactIds),
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

  protected async toggleConfigEnabled(config: ArtifactNotificationConfiguration) {
    this.enabledToggleLoading.set(true);
    try {
      const enabled = !config.enabled;
      await firstValueFrom(
        this.svc.update(config.id, {
          name: config.name,
          enabled,
          newVersionTriggerEnabled: config.newVersionTriggerEnabled,
          artifactIds: config.artifacts.map((it) => it.id),
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

  protected async deleteConfig(config: ArtifactNotificationConfiguration) {
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
