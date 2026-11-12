import {Component, computed, inject, signal} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {AbstractControl, FormBuilder, ReactiveFormsModule, ValidationErrors, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPen, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, map} from 'rxjs';
import {getFormDisplayedError} from '../../../../util/errors';
import {slugMaxLength, slugPattern, toSlug} from '../../../../util/slug';
import {AutotrimDirective} from '../../../directives/autotrim.directive';
import {CustomOidcService} from '../../../services/custom-oidc.service';
import {OrganizationService} from '../../../services/organization.service';
import {OverlayData} from '../../../services/overlay.service';
import {ToastService} from '../../../services/toast.service';
import {CustomDomain} from '../../../types/custom-domain';
import {CustomOidcConfiguration} from '../../../types/custom-oidc';
import {ClipComponent} from '../../clip.component';
import {ClosableDialog} from '../../confirm-dialog/closable-dialog';

const DEFAULT_SCOPES = ['openid', 'profile', 'email'];

export interface IdentityProviderModalData {
  domain: CustomDomain;
  customerOrganizationId?: string;
  configuration?: CustomOidcConfiguration;
}

@Component({
  templateUrl: './identity-provider-modal.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, AutotrimDirective, ClipComponent],
})
export class IdentityProviderModalComponent extends ClosableDialog<boolean> {
  protected readonly faPen = faPen;
  protected readonly faXmark = faXmark;

  private readonly customOidcService = inject(CustomOidcService);
  private readonly organizationService = inject(OrganizationService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly data = inject(OverlayData) as IdentityProviderModalData;

  protected readonly editing = this.data.configuration;
  protected readonly provisioningAvailable =
    this.data.domain.domainType !== 'customer_portal' || !!this.data.domain.customerOrganizationId;
  protected readonly saving = signal(false);

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
      clientSecret: this.fb.control('', this.editing ? [] : [Validators.required]),
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

  private readonly organizationSlug = toSignal(
    this.organizationService.get().pipe(map((organization) => organization.slug)),
    {initialValue: undefined}
  );
  protected readonly slugValue = toSignal(this.form.controls.slug.valueChanges, {initialValue: ''});
  protected readonly callbackUrlPrefix = computed(() => {
    const organizationSlug = this.organizationSlug();
    return organizationSlug
      ? `${location.protocol}//${this.data.domain.domain}/api/v1/auth/oidc/custom/${organizationSlug}/`
      : undefined;
  });
  protected readonly callbackUrl = computed(() => {
    const prefix = this.callbackUrlPrefix();
    const slug = this.slugValue();
    return prefix && slug ? `${prefix}${slug}/callback` : undefined;
  });

  private readonly slugEdited = signal(!!this.editing);
  protected readonly editingSlug = signal(false);

  constructor() {
    super();
    this.form.controls.name.valueChanges.pipe(takeUntilDestroyed()).subscribe((name) => {
      if (!this.slugEdited()) {
        this.form.controls.slug.setValue(toSlug(name));
      }
    });
    if (this.editing) {
      this.form.setValue({
        name: this.editing.name,
        slug: this.editing.slug,
        issuer: this.editing.issuer,
        clientId: this.editing.clientId,
        clientSecret: '',
        spInitiated: this.editing.spInitiated,
        createUnknownUsers: this.editing.createUnknownUsers && this.provisioningAvailable,
        defaultUserRole: this.editing.defaultUserRole,
        allowedEmailDomains: this.editing.allowedEmailDomains.join(' '),
      });
    }
  }

  protected editSlug() {
    this.slugEdited.set(true);
    this.editingSlug.set(true);
  }

  protected async save() {
    this.form.markAllAsTouched();
    if (!this.form.valid) {
      return;
    }
    const value = this.form.getRawValue();
    const request = {
      customDomainId: this.data.domain.id,
      customerOrganizationId: this.data.customerOrganizationId,
      name: value.name,
      slug: value.slug,
      enabled: this.editing?.enabled ?? true,
      issuer: value.issuer,
      clientId: value.clientId,
      clientSecret: value.clientSecret || undefined,
      scopes: this.editing?.scopes ?? DEFAULT_SCOPES,
      pkceEnabled: this.editing?.pkceEnabled,
      spInitiated: value.spInitiated,
      createUnknownUsers: value.createUnknownUsers,
      defaultUserRole: value.defaultUserRole,
      allowedEmailDomains: splitList(value.allowedEmailDomains),
    };

    this.saving.set(true);
    try {
      if (this.editing) {
        await firstValueFrom(this.customOidcService.update(this.editing.id, request));
      } else {
        await firstValueFrom(this.customOidcService.create(request));
      }
      this.toast.success(this.editing ? 'Identity provider updated' : 'Identity provider created');
      await this.dialogRef.close(true);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.saving.set(false);
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
