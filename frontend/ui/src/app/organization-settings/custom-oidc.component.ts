import {ChangeDetectionStrategy, Component, computed, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {AbstractControl, FormBuilder, ReactiveFormsModule, ValidationErrors, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleExclamation, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {BehaviorSubject, combineLatest, firstValueFrom, of, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {ClipComponent} from '../components/clip.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {CustomDomainsService} from '../services/custom-domains.service';
import {CustomOidcService} from '../services/custom-oidc.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {CustomDomain} from '../types/custom-domain';
import {CustomOidcConfiguration, CustomOidcConfigurationsResponse} from '../types/custom-oidc';

@Component({
  selector: 'app-custom-oidc',
  templateUrl: './custom-oidc.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [FaIconComponent, ReactiveFormsModule, AutotrimDirective, ClipComponent],
})
export class CustomOidcComponent {
  protected readonly faPlus = faPlus;
  protected readonly faPen = faPen;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly faCircleExclamation = faCircleExclamation;

  private readonly customOidcService = inject(CustomOidcService);
  private readonly customDomainsService = inject(CustomDomainsService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(FormBuilder).nonNullable;

  private readonly refresh$ = new BehaviorSubject<void>(undefined);

  protected readonly visible = computed(
    () => this.featureFlags.isCustomOidcProvidersEnabled() && this.auth.isVendor() && this.auth.hasRole('admin')
  );

  private readonly response = toSignal(
    combineLatest([this.featureFlags.isCustomOidcProvidersEnabled$, this.refresh$]).pipe(
      switchMap(([enabled]) =>
        enabled
          ? this.customOidcService.list()
          : of({configurations: [], membersWithOtherOrganizations: 0} as CustomOidcConfigurationsResponse)
      )
    ),
    {initialValue: {configurations: [], membersWithOtherOrganizations: 0} as CustomOidcConfigurationsResponse}
  );
  protected readonly configurations = computed(() => this.response().configurations);
  protected readonly membersWithOtherOrganizations = computed(() => this.response().membersWithOtherOrganizations);

  private readonly domains = toSignal(
    combineLatest([this.featureFlags.isCustomOidcProvidersEnabled$, this.refresh$]).pipe(
      switchMap(([enabled]) => (enabled ? this.customDomainsService.list() : of([] as CustomDomain[])))
    ),
    {initialValue: [] as CustomDomain[]}
  );
  protected readonly appDomain = computed(() => this.domains().find((domain) => domain.domainType === 'app'));

  protected readonly editing = signal<CustomOidcConfiguration | undefined>(undefined);
  protected readonly saving = signal(false);
  private readonly formDialog = viewChild.required<TemplateRef<unknown>>('formDialog');
  private dialogRef?: DialogRef;

  protected readonly form = this.fb.group(
    {
      name: this.fb.control('', [Validators.required, Validators.maxLength(100)]),
      issuer: this.fb.control('', [Validators.required, Validators.pattern(/^https?:\/\/\S+$/)]),
      clientId: this.fb.control('', [Validators.required]),
      clientSecret: this.fb.control(''),
      scopes: this.fb.control('profile email'),
      pkce: this.fb.control<'auto' | 'on' | 'off'>('auto'),
      enabled: this.fb.control(true),
      spInitiated: this.fb.control(false),
      createUnknownUsers: this.fb.control(false),
      defaultUserRole: this.fb.control<'read_only' | 'read_write' | 'admin'>('read_write'),
      allowedEmailDomains: this.fb.control(''),
    },
    {validators: [provisioningNeedsEmailDomains]}
  );

  protected showDialog(configuration?: CustomOidcConfiguration) {
    this.editing.set(configuration);
    if (configuration) {
      this.form.setValue({
        name: configuration.name,
        issuer: configuration.issuer,
        clientId: configuration.clientId,
        clientSecret: '',
        scopes: configuration.scopes.filter((scope) => scope !== 'openid').join(' '),
        pkce: configuration.pkceEnabled === undefined ? 'auto' : configuration.pkceEnabled ? 'on' : 'off',
        enabled: configuration.enabled,
        spInitiated: configuration.spInitiated,
        createUnknownUsers: configuration.createUnknownUsers,
        defaultUserRole: configuration.defaultUserRole,
        allowedEmailDomains: configuration.allowedEmailDomains.join(' '),
      });
    } else {
      this.form.reset();
    }
    this.dialogRef = this.overlay.showModal(this.formDialog());
  }

  protected closeDialog() {
    this.form.reset();
    this.editing.set(undefined);
    this.dialogRef?.close();
  }

  protected async save() {
    this.form.markAllAsTouched();
    const customDomainId = this.appDomain()?.id;
    if (!this.form.valid || !customDomainId) {
      return;
    }
    const value = this.form.getRawValue();
    const existing = this.editing();
    const request = {
      customDomainId,
      name: value.name,
      enabled: value.enabled,
      issuer: value.issuer,
      clientId: value.clientId,
      clientSecret: value.clientSecret || undefined,
      scopes: splitList(value.scopes),
      pkceEnabled: value.pkce === 'auto' ? undefined : value.pkce === 'on',
      spInitiated: value.spInitiated,
      createUnknownUsers: value.createUnknownUsers,
      defaultUserRole: value.defaultUserRole,
      allowedEmailDomains: splitList(value.allowedEmailDomains),
    };

    this.saving.set(true);
    try {
      if (existing) {
        await firstValueFrom(this.customOidcService.update(existing.id, request));
      } else {
        await firstValueFrom(this.customOidcService.create(request));
      }
      this.toast.success(existing ? 'Identity provider updated' : 'Identity provider created');
      this.refresh$.next();
      this.closeDialog();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.saving.set(false);
    }
  }

  protected async test(configuration: CustomOidcConfiguration) {
    try {
      await firstValueFrom(this.customOidcService.test(configuration.id));
      this.toast.success(`${configuration.name} serves a valid OpenID configuration`);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected async remove(configuration: CustomOidcConfiguration) {
    const confirmed = await firstValueFrom(
      this.overlay.confirm(
        `Really delete ${configuration.name}? Everyone who signs in through it loses that login, and users without ` +
          `a password will have to reset it.`
      )
    );
    if (!confirmed) {
      return;
    }
    try {
      await firstValueFrom(this.customOidcService.delete(configuration.id));
      this.refresh$.next();
      this.toast.success('Identity provider deleted');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}

function provisioningNeedsEmailDomains(group: AbstractControl): ValidationErrors | null {
  const value = group.value as {createUnknownUsers?: boolean; allowedEmailDomains?: string};
  return value.createUnknownUsers && splitList(value.allowedEmailDomains ?? '').length === 0
    ? {emailDomainsRequired: true}
    : null;
}

function splitList(value: string): string[] {
  return value
    .split(/[\s,]+/)
    .map((entry) => entry.trim())
    .filter((entry) => entry.length > 0);
}
