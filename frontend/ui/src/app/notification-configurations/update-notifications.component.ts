import {DatePipe} from '@angular/common';
import {Component, computed, inject, input, signal, TemplateRef, viewChild} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {AbstractControl, FormBuilder, ReactiveFormsModule, ValidationErrors, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, firstValueFrom, map, of, shareReplay, startWith, Subject, switchMap} from 'rxjs';
import {compareBy} from '../../util/arrays';
import {getFormDisplayedError} from '../../util/errors';
import {checkedIds, checkedRecord} from '../../util/formRecord';
import {validateRecordAtLeast} from '../../util/validation';
import {ApplicationLogoComponent, ApplicationPreviewComponent} from '../applications/components';
import {ArtifactLogoComponent, ArtifactPreviewComponent} from '../artifacts/components';
import {PillTabBarComponent} from '../components/pill-tab-bar.component';
import {SearchBarComponent} from '../components/search-bar.component';
import {TabItem} from '../components/tab-bar.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ApplicationEntitlementsService} from '../services/application-entitlements.service';
import {ApplicationsService} from '../services/applications.service';
import {ArtifactEntitlementsService} from '../services/artifact-entitlements.service';
import {ArtifactsService} from '../services/artifacts.service';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {CustomerOrganizationsCache} from '../services/customer-organizations.service';
import {UpdateNotificationConfigurationsService} from '../services/notification-configurations.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {UsersService} from '../services/users.service';
import {UpdateNotificationConfiguration} from '../types/notification-configuration';
import {NotificationRecipientsComponent} from './notification-recipients.component';

/** The two kinds of version a configuration can watch, which are the sections of the picklist. */
type NotificationKind = 'application' | 'artifact';

@Component({
  selector: 'app-update-notifications',
  templateUrl: './update-notifications.component.html',
  providers: [CustomerOrganizationsCache],
  imports: [
    FaIconComponent,
    ReactiveFormsModule,
    DatePipe,
    PillTabBarComponent,
    SearchBarComponent,
    AutotrimDirective,
    ApplicationLogoComponent,
    ApplicationPreviewComponent,
    ArtifactLogoComponent,
    ArtifactPreviewComponent,
    NotificationRecipientsComponent,
  ],
})
export class UpdateNotificationsComponent {
  /** customerOrganizationId scopes the page to what one customer is notified about. */
  public readonly customerOrganizationId = input<string>();

  protected readonly auth = inject(AuthService);
  private readonly configsService = inject(UpdateNotificationConfigurationsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly artifactsService = inject(ArtifactsService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly artifactEntitlementsService = inject(ArtifactEntitlementsService);
  private readonly usersService = inject(UsersService);
  private readonly context = inject(ContextService);
  private readonly customerOrganizations = inject(CustomerOrganizationsCache);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  protected readonly faPlus = faPlus;
  protected readonly faPen = faPen;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;

  private readonly customerOrganizationId$ = toObservable(this.customerOrganizationId);

  /** The customer a configuration of this page notifies, which is the caller on a customer's own page. */
  private readonly scopedCustomer$ = this.customerOrganizationId$.pipe(
    switchMap((customerOrganizationId) =>
      customerOrganizationId
        ? this.customerOrganizations
            .getCustomerOrganizations()
            .pipe(map((customers) => customers.find((it) => it.id === customerOrganizationId)))
        : this.context.getCustomerOrganization()
    ),
    shareReplay(1)
  );

  // A customer watches only what it can reach: applications through its deployments, artifacts
  // through the registry.
  private readonly kinds$ = combineLatest([this.customerOrganizationId$, this.scopedCustomer$]).pipe(
    map(([customerOrganizationId, customer]): NotificationKind[] => {
      if (!this.auth.isCustomer() && !customerOrganizationId) {
        return ['application', 'artifact'];
      }
      const features = customer?.features ?? [];
      return [
        ...(features.includes('deployment_targets') ? (['application'] as const) : []),
        ...(features.includes('artifacts') ? (['artifact'] as const) : []),
      ];
    }),
    shareReplay(1)
  );
  protected readonly kinds = toSignal(this.kinds$, {initialValue: [] as NotificationKind[]});

  private readonly reload$ = new Subject<void>();
  private readonly reloads$ = this.reload$.pipe(startWith(undefined));

  private readonly configs = toSignal(
    combineLatest([this.reloads$, this.customerOrganizationId$]).pipe(
      switchMap(([, customerOrganizationId]) => this.configsService.list(customerOrganizationId))
    ),
    {initialValue: [] as UpdateNotificationConfiguration[]}
  );
  private readonly rows = computed(() => [...this.configs()].sort(compareBy((config) => config.name)));

  private readonly unsortedApplications = toSignal(
    this.kinds$.pipe(switchMap((kinds) => (kinds.includes('application') ? this.applicationsService.list() : of([])))),
    {initialValue: []}
  );
  protected readonly applications = computed(() => {
    const entitled = this.entitledApplicationIds();
    return [...this.unsortedApplications()]
      .filter((application) => entitled === undefined || entitled.has(application.id!))
      .sort(compareBy((application) => application.name!));
  });
  private readonly unsortedArtifacts = toSignal(
    this.kinds$.pipe(switchMap((kinds) => (kinds.includes('artifact') ? this.artifactsService.list() : of([])))),
    {initialValue: []}
  );
  protected readonly artifacts = computed(() => {
    const entitled = this.entitledArtifactIds();
    return [...this.unsortedArtifacts()]
      .filter((artifact) => entitled === undefined || entitled.has(artifact.id))
      .sort(compareBy((artifact) => artifact.name));
  });

  // What a vendor may set up for a customer is what that customer is entitled to. The server drops
  // anything else from the configuration, which would look like the selection had not been saved.
  private readonly applicationEntitlements = toSignal(
    this.customerOrganizationId$.pipe(
      switchMap((customerOrganizationId) =>
        customerOrganizationId && !this.auth.isCustomer() ? this.applicationEntitlementsService.list() : of(undefined)
      )
    )
  );
  private readonly artifactEntitlements = toSignal(
    this.customerOrganizationId$.pipe(
      switchMap((customerOrganizationId) =>
        customerOrganizationId && !this.auth.isCustomer() ? this.artifactEntitlementsService.list() : of(undefined)
      )
    )
  );
  private readonly entitledApplicationIds = computed(() => {
    const entitlements = this.applicationEntitlements();
    return entitlements === undefined
      ? undefined
      : new Set(
          entitlements
            .filter((it) => it.customerOrganizationId === this.customerOrganizationId() && !isExpired(it.expiresAt))
            .map((it) => it.applicationId!)
        );
  });
  private readonly entitledArtifactIds = computed(() => {
    const entitlements = this.artifactEntitlements();
    return entitlements === undefined
      ? undefined
      : new Set(
          entitlements
            .filter((it) => it.customerOrganizationId === this.customerOrganizationId() && !isExpired(it.expiresAt))
            .flatMap((it) => (it.artifacts ?? []).map((artifact) => artifact.artifactId))
        );
  });

  private readonly allUsers = toSignal(this.usersService.getUsers(), {initialValue: []});
  protected readonly users = computed(() => {
    const customerOrganizationId = this.customerOrganizationId();
    return customerOrganizationId
      ? this.allUsers().filter((it) => it.customerOrganizationId === customerOrganizationId)
      : this.allUsers();
  });

  protected readonly filterForm = this.fb.group({search: ''});
  private readonly filterValue = toSignal(this.filterForm.controls.search.valueChanges);
  protected readonly filteredRows = computed(() => {
    const value = this.filterValue()?.toLowerCase();
    const rows = this.rows();
    return !value ? rows : rows.filter((config) => config.name.toLowerCase().includes(value));
  });

  protected readonly editFormLoading = signal(false);
  protected readonly enabledToggleLoading = signal(false);

  protected readonly editConfigForm = this.fb.group(
    {
      id: this.fb.control(''),
      name: this.fb.control('', [Validators.required]),
      enabled: this.fb.control(true),
      applicationIds: this.fb.record<boolean>({}),
      artifactIds: this.fb.record<boolean>({}),
      userAccountIds: this.fb.record<boolean>({}, {validators: [validateRecordAtLeast(1)]}),
    },
    {validators: [validateAnyTarget]}
  );

  protected readonly picklistTab = signal<NotificationKind>('application');
  protected readonly picklistTabs = computed<TabItem<NotificationKind>[]>(() =>
    this.kinds().map((kind) => ({id: kind, label: kind === 'application' ? 'Applications' : 'Artifacts'}))
  );

  private readonly editConfigDrawerTpl = viewChild.required<TemplateRef<unknown>>('editConfigDrawer');
  private editConfigDrawerRef?: DialogRef;

  constructor() {
    // Warms the cache the recipient list reads, which it would otherwise fill on its first render.
    if (this.auth.isVendor()) {
      this.customerOrganizations.getCustomerOrganizations().pipe(takeUntilDestroyed()).subscribe();
    }
  }

  protected async showDrawer(config?: UpdateNotificationConfiguration) {
    this.hideDrawer();
    this.editConfigForm.reset();

    const kinds = this.kinds();
    this.picklistTab.set(this.defaultTab(kinds, config));
    this.addPicklistControls(config);

    if (config) {
      this.editConfigForm.patchValue({
        id: config.id,
        name: config.name,
        enabled: config.enabled,
        applicationIds: checkedRecord(config.applications.map((it) => it.id)),
        artifactIds: checkedRecord(config.artifacts.map((it) => it.id)),
        userAccountIds: checkedRecord(config.recipients.map((it) => it.id)),
      });
    }

    const drawerRef = (this.editConfigDrawerRef = this.overlay.showDrawer(this.editConfigDrawerTpl()));

    // The drawer shows the lists the page has loaded already. The refresh runs behind it and only
    // adds what is missing, so a box ticked in the meantime survives it.
    await Promise.all([
      kinds.includes('application') ? this.applicationsService.refresh() : Promise.resolve(),
      kinds.includes('artifact') ? this.artifactsService.refresh() : Promise.resolve(),
      this.usersService.refresh(),
    ]);
    if (this.editConfigDrawerRef === drawerRef) {
      this.addPicklistControls(config);
    }
  }

  /** The section the drawer opens on, which is the one that has something to show. */
  private defaultTab(kinds: NotificationKind[], config?: UpdateNotificationConfiguration): NotificationKind {
    if (config && config.applications.length === 0 && config.artifacts.length > 0) {
      return 'artifact';
    }
    return (
      kinds.find((kind) =>
        kind === 'application'
          ? this.applications().length > 0
          : this.artifacts().some((artifact) => !artifact.upstreamUrl)
      ) ??
      kinds[0] ??
      'application'
    );
  }

  /** Adds a checkbox for every entry known now, checked where the edited configuration uses it. */
  private addPicklistControls(config?: UpdateNotificationConfiguration) {
    const checked = new Set([
      ...(config?.applications ?? []).map((it) => it.id),
      ...(config?.artifacts ?? []).map((it) => it.id),
      ...(config?.recipients ?? []).map((it) => it.id),
    ]);
    const controls = this.editConfigForm.controls;
    for (const application of this.applications()) {
      controls.applicationIds.addControl(application.id!, this.fb.control(checked.has(application.id!)));
    }
    for (const artifact of this.artifacts()) {
      const control = this.fb.control(checked.has(artifact.id));
      // A mirrored artifact cannot be watched: every version of one appears at once when the
      // upstream sync runs, so announcing them would mean a notification per upstream tag.
      if (artifact.upstreamUrl) {
        control.disable();
      }
      controls.artifactIds.addControl(artifact.id, control);
    }
    for (const user of this.users()) {
      controls.userAccountIds.addControl(user.id!, this.fb.control(checked.has(user.id!)));
    }
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
    this.editFormLoading.set(true);
    try {
      const request = {
        customerOrganizationId: this.customerOrganizationId(),
        name: formValue.name,
        enabled: formValue.enabled,
        applicationIds: checkedIds(formValue.applicationIds),
        artifactIds: checkedIds(formValue.artifactIds),
        userAccountIds: checkedIds(formValue.userAccountIds),
      };
      await firstValueFrom(
        formValue.id ? this.configsService.update(formValue.id, request) : this.configsService.create(request)
      );
      this.toast.success(`Notification configuration ${formValue.id ? 'updated' : 'created'}`);
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

  protected async toggleConfigEnabled(config: UpdateNotificationConfiguration) {
    this.enabledToggleLoading.set(true);
    try {
      const enabled = !config.enabled;
      await firstValueFrom(
        this.configsService.update(config.id, {
          customerOrganizationId: this.customerOrganizationId(),
          name: config.name,
          enabled,
          applicationIds: config.applications.map((it) => it.id),
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

  protected async deleteConfig(config: UpdateNotificationConfiguration) {
    if (!(await firstValueFrom(this.overlay.confirm(`Really delete the notification "${config.name}"?`)))) {
      return;
    }

    try {
      await firstValueFrom(this.configsService.delete(config.id, this.customerOrganizationId()));
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

function isExpired(expiresAt?: Date): boolean {
  return expiresAt !== undefined && new Date(expiresAt).getTime() <= Date.now();
}

function validateAnyTarget(control: AbstractControl): ValidationErrors | null {
  const value = control.value as {
    applicationIds: Record<string, boolean>;
    artifactIds: Record<string, boolean>;
  };
  const checked = [...checkedIds(value.applicationIds), ...checkedIds(value.artifactIds)];
  return checked.length > 0 ? null : {noTargets: 1};
}
