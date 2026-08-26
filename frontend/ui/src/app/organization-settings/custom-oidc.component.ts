import {NgTemplateOutlet} from '@angular/common';
import {Component, computed, inject, input, linkedSignal, signal, TemplateRef, viewChild} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {AbstractControl, FormBuilder, ReactiveFormsModule, ValidationErrors, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPen, faPlug, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {BehaviorSubject, combineLatest, firstValueFrom, from, map, of, switchMap} from 'rxjs';
import {getRemoteEnvironment} from '../../env/remote';
import {getFormDisplayedError} from '../../util/errors';
import {slugMaxLength, slugPattern, toSlug} from '../../util/slug';
import {ClipComponent} from '../components/clip.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {CustomDomainsService} from '../services/custom-domains.service';
import {CustomOidcService} from '../services/custom-oidc.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {OrganizationService} from '../services/organization.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {CustomDomain, CustomDomainType, CustomDomainVerification} from '../types/custom-domain';
import {CustomOidcConfiguration, CustomOidcConfigurationsResponse} from '../types/custom-oidc';
import {DomainFieldComponent} from './domain-field.component';

const BUSINESS_OIDC_BANNER_DISMISSED_KEY = 'customOidc.businessOidcBannerDismissed';

const DEFAULT_SCOPES = ['openid', 'profile', 'email'];

@Component({
  selector: 'app-custom-oidc',
  templateUrl: './custom-oidc.component.html',
  imports: [
    FaIconComponent,
    ReactiveFormsModule,
    AutotrimDirective,
    ClipComponent,
    NgTemplateOutlet,
    DomainFieldComponent,
    RouterLink,
  ],
})
export class CustomOidcComponent {
  protected readonly faPlus = faPlus;
  protected readonly faPen = faPen;
  protected readonly faPlug = faPlug;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;

  private readonly customOidcService = inject(CustomOidcService);
  private readonly customDomainsService = inject(CustomDomainsService);
  private readonly organizationService = inject(OrganizationService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly contextService = inject(ContextService);
  private readonly fb = inject(FormBuilder).nonNullable;

  private readonly remoteEnv = toSignal(from(getRemoteEnvironment()));
  protected readonly cnameTarget = computed(() => this.remoteEnv()?.customDomainTarget);

  private readonly domainsRefresh$ = new BehaviorSubject<void>(undefined);
  private readonly configurationsRefresh$ = new BehaviorSubject<void>(undefined);

  public readonly customerOrganizationId = input<string>();

  protected readonly customerScoped = computed(() => !!this.customerOrganizationId());
  protected readonly viewerIsCustomer = computed(() => this.auth.isCustomer());
  protected readonly viewerIsVendor = computed(() => this.auth.isVendor());
  protected readonly partnerManagementEnabled = computed(() => this.featureFlags.isPartnerManagementEnabled());
  protected readonly appDomainLabel = computed(() =>
    this.partnerManagementEnabled() ? 'Vendor & Partner Portal domain' : 'Vendor Portal domain'
  );
  protected readonly customerPortalCheckboxLabel = computed(() =>
    this.customerScoped() ? undefined : 'Use a customer portal domain'
  );
  protected readonly customDomainsConfigured = computed(() => !!this.cnameTarget());

  protected readonly domainsEnabled = computed(() => this.featureFlags.isCustomDomainsEnabled());
  protected readonly oidcProvidersEnabled = computed(() => this.featureFlags.isCustomOidcProvidersEnabled());

  private readonly businessOidcBannerDismissed = signal(
    sessionStorage.getItem(BUSINESS_OIDC_BANNER_DISMISSED_KEY) === 'true'
  );
  protected readonly showBusinessOidcBanner = computed(
    () =>
      !this.customerScoped() &&
      this.domainsEnabled() &&
      !this.oidcProvidersEnabled() &&
      !this.businessOidcBannerDismissed()
  );

  protected dismissBusinessOidcBanner(): void {
    sessionStorage.setItem(BUSINESS_OIDC_BANNER_DISMISSED_KEY, 'true');
    this.businessOidcBannerDismissed.set(true);
  }

  protected readonly visible = computed(
    () =>
      (this.domainsEnabled() || this.oidcProvidersEnabled()) &&
      (this.customerScoped() || this.auth.isVendor()) &&
      this.auth.hasRole('admin')
  );

  private readonly response = toSignal(
    combineLatest([this.featureFlags.isCustomOidcProvidersEnabled$, this.configurationsRefresh$]).pipe(
      switchMap(([enabled]) =>
        enabled ? this.customOidcService.list() : of({configurations: []} as CustomOidcConfigurationsResponse)
      )
    ),
    {initialValue: {configurations: []} as CustomOidcConfigurationsResponse}
  );
  protected readonly configurations = computed(() => this.response().configurations);

  private readonly fetchedDomains = toSignal(
    combineLatest([
      this.featureFlags.isCustomDomainsEnabled$,
      this.featureFlags.isCustomOidcProvidersEnabled$,
      this.domainsRefresh$,
    ]).pipe(
      switchMap(([domainsEnabled, oidcEnabled]) =>
        domainsEnabled || oidcEnabled ? this.customDomainsService.list() : of([] as CustomDomain[])
      )
    ),
    {initialValue: [] as CustomDomain[]}
  );
  private readonly domains = linkedSignal(() => this.fetchedDomains());
  protected readonly scopedDomains = computed(() =>
    this.domains().filter((d) => (d.customerOrganizationId ?? undefined) === this.customerOrganizationId())
  );
  protected readonly appDomain = computed(() => this.scopedDomains().find((domain) => domain.domainType === 'app'));
  protected readonly registryDomain = computed(() =>
    this.scopedDomains().find((domain) => domain.domainType === 'registry')
  );
  protected readonly customerPortalDomain = computed(() =>
    this.scopedDomains().find((domain) => domain.domainType === 'customer_portal')
  );

  protected readonly checkingDomainIds = signal<ReadonlySet<string>>(new Set());

  constructor() {
    this.form.controls.name.valueChanges.pipe(takeUntilDestroyed()).subscribe((name) => {
      if (!this.slugEdited()) {
        this.form.controls.slug.setValue(toSlug(name));
      }
    });
  }

  protected async verifyDomain(domain: CustomDomain) {
    this.checkingDomainIds.update((ids) => new Set(ids).add(domain.id));
    try {
      const verification = await firstValueFrom(this.customDomainsService.verify(domain.id));
      this.applyVerification(verification);
      if (verification.inconclusive) {
        this.toast.error('The DNS lookup could not be completed, please try again');
      } else if (verification.verified !== domain.verified) {
        this.contextService.reload();
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.checkingDomainIds.update((ids) => {
        const next = new Set(ids);
        next.delete(domain.id);
        return next;
      });
    }
  }

  private applyVerification(verification: CustomDomainVerification) {
    this.domains.update((domains) =>
      domains.map((domain) =>
        domain.id === verification.customDomainId
          ? {
              ...domain,
              verified: verification.verified,
              verifiedAt: verification.verifiedAt,
              verificationCheckedAt: verification.verificationCheckedAt,
              verificationError: verification.verificationError,
            }
          : domain
      )
    );
  }

  protected readonly savingDomain = signal(false);

  protected async saveDomain(value: string, domainType: CustomDomainType) {
    this.savingDomain.set(true);
    try {
      await firstValueFrom(
        this.customDomainsService.create([{domain: value, domainType}], this.customerOrganizationId())
      );
      this.domainsRefresh$.next();
      this.contextService.reload();
      this.toast.success('Custom domain saved');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.savingDomain.set(false);
    }
  }

  protected async removeDomain(domain: CustomDomain | undefined) {
    if (!domain) {
      return;
    }
    const confirmed = await firstValueFrom(
      this.overlay.confirm(
        `Really remove ${domain.domain}? The domain will stop working and its TLS certificate will no longer be renewed.`
      )
    );
    if (!confirmed) {
      return;
    }
    try {
      await firstValueFrom(this.customDomainsService.delete(domain.id));
      this.domainsRefresh$.next();
      this.configurationsRefresh$.next();
      this.contextService.reload();
      this.toast.success('Custom domain removed');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected readonly organizationSlug = toSignal(
    this.organizationService.get().pipe(map((organization) => organization.slug)),
    {initialValue: undefined}
  );

  protected readonly configurable = computed(() => !!this.organizationSlug());

  protected configurationsFor(domain: CustomDomain | undefined): CustomOidcConfiguration[] {
    return domain ? this.configurations().filter((configuration) => configuration.customDomainId === domain.id) : [];
  }

  private readonly activeDomain = signal<CustomDomain | undefined>(undefined);
  protected readonly editing = signal<CustomOidcConfiguration | undefined>(undefined);
  protected readonly saving = signal(false);
  protected readonly toggling = signal<string | undefined>(undefined);
  private readonly formDialog = viewChild.required<TemplateRef<unknown>>('formDialog');
  private dialogRef?: DialogRef;

  protected readonly form = this.fb.group(
    {
      name: this.fb.control('', [Validators.required, Validators.maxLength(100)]),
      slug: this.fb.control('', [
        Validators.required,
        Validators.pattern(slugPattern),
        Validators.maxLength(slugMaxLength),
      ]),
      issuer: this.fb.control('', [Validators.required, Validators.pattern(/^https?:\/\/\S+$/)]),
      clientId: this.fb.control('', [Validators.required]),
      clientSecret: this.fb.control(''),
      spInitiated: this.fb.control(false),
      createUnknownUsers: this.fb.control(false),
      defaultUserRole: this.fb.control<'read_only' | 'read_write' | 'admin'>('read_write'),
      allowedEmailDomains: this.fb.control(''),
    },
    {validators: [provisioningNeedsEmailDomains]}
  );

  protected readonly createUnknownUsers = toSignal(this.form.controls.createUnknownUsers.valueChanges, {
    initialValue: false,
  });

  protected readonly slugValue = toSignal(this.form.controls.slug.valueChanges, {initialValue: ''});
  protected readonly callbackUrlPrefix = computed(() => {
    const domain = this.activeDomain()?.domain;
    const organizationSlug = this.organizationSlug();
    if (!domain || !organizationSlug) {
      return undefined;
    }
    return `${location.protocol}//${domain}/api/v1/auth/oidc/custom/${organizationSlug}/`;
  });
  protected readonly callbackUrl = computed(() => {
    const prefix = this.callbackUrlPrefix();
    const slug = this.slugValue();
    return prefix && slug ? `${prefix}${slug}/callback` : undefined;
  });

  private readonly slugEdited = signal(false);
  protected readonly editingSlug = signal(false);

  protected editSlug() {
    this.slugEdited.set(true);
    this.editingSlug.set(true);
  }

  protected showDialog(domain: CustomDomain, configuration?: CustomOidcConfiguration) {
    this.activeDomain.set(domain);
    this.editing.set(configuration);
    this.slugEdited.set(!!configuration);
    this.editingSlug.set(false);
    this.form.controls.clientSecret.setValidators(configuration ? [] : [Validators.required]);
    if (configuration) {
      this.form.setValue({
        name: configuration.name,
        slug: configuration.slug,
        issuer: configuration.issuer,
        clientId: configuration.clientId,
        clientSecret: '',
        spInitiated: configuration.spInitiated,
        createUnknownUsers: configuration.createUnknownUsers && this.provisioningAvailable(),
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
    this.activeDomain.set(undefined);
    this.dialogRef?.close();
  }

  protected readonly provisioningAvailable = computed(() => {
    const domain = this.activeDomain();
    return !!domain && (domain.domainType !== 'customer_portal' || !!domain.customerOrganizationId);
  });

  protected async save() {
    this.form.markAllAsTouched();
    const customDomainId = this.activeDomain()?.id;
    if (!this.form.valid || !customDomainId) {
      return;
    }
    const value = this.form.getRawValue();
    const existing = this.editing();
    const request = {
      customDomainId,
      customerOrganizationId: this.customerOrganizationId(),
      name: value.name,
      slug: value.slug,
      enabled: existing?.enabled ?? true,
      issuer: value.issuer,
      clientId: value.clientId,
      clientSecret: value.clientSecret || undefined,
      scopes: existing?.scopes ?? DEFAULT_SCOPES,
      pkceEnabled: existing?.pkceEnabled,
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
      this.configurationsRefresh$.next();
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

  protected async toggleEnabled(configuration: CustomOidcConfiguration, toggle: HTMLInputElement) {
    this.toggling.set(configuration.id);
    try {
      await firstValueFrom(
        this.customOidcService.update(configuration.id, {
          customDomainId: configuration.customDomainId,
          name: configuration.name,
          slug: configuration.slug,
          enabled: !configuration.enabled,
          issuer: configuration.issuer,
          clientId: configuration.clientId,
          scopes: configuration.scopes,
          pkceEnabled: configuration.pkceEnabled,
          spInitiated: configuration.spInitiated,
          createUnknownUsers: configuration.createUnknownUsers,
          defaultUserRole: configuration.defaultUserRole,
          allowedEmailDomains: configuration.allowedEmailDomains,
        })
      );
      this.configurationsRefresh$.next();
    } catch (e) {
      toggle.checked = configuration.enabled;
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.toggling.set(undefined);
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
      this.configurationsRefresh$.next();
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
