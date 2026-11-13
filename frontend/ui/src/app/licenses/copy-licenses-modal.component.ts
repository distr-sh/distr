import {DatePipe} from '@angular/common';
import {Component, computed, effect, inject, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faXmark} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {firstValueFrom, forkJoin} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {ClosableDialog} from '../components/confirm-dialog/closable-dialog';
import {ExpiresAtPickerComponent} from '../components/expires-at-picker/expires-at-picker.component';
import {SpinnerComponent} from '../components/spinner/spinner.component';
import {ApplicationEntitlementsService} from '../services/application-entitlements.service';
import {ArtifactEntitlementsService} from '../services/artifact-entitlements.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {LicenseKeysService} from '../services/license-keys.service';
import {OverlayData} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {License} from '../types/license';

export interface CopyLicensesContext {
  targetCustomerOrganizationId: string;
}

/** Copies every license key and entitlement of another customer to the target customer. Closes with true once copied. */
@Component({
  templateUrl: './copy-licenses-modal.component.html',
  imports: [DatePipe, ReactiveFormsModule, FaIconComponent, ExpiresAtPickerComponent, SpinnerComponent],
})
export class CopyLicensesModalComponent extends ClosableDialog<boolean> {
  private readonly targetCustomerOrganizationId = (inject(OverlayData) as CopyLicensesContext)
    .targetCustomerOrganizationId;
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly licenseKeysService = inject(LicenseKeysService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly artifactEntitlementsService = inject(ArtifactEntitlementsService);
  private readonly toast = inject(ToastService);

  protected readonly faXmark = faXmark;

  protected readonly loading = signal(true);
  protected readonly copying = signal(false);

  private readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations(), {
    initialValue: [],
  });
  private readonly licenseKeys = toSignal(this.licenseKeysService.list(), {initialValue: []});
  private readonly applicationEntitlements = toSignal(this.applicationEntitlementsService.list(), {initialValue: []});
  private readonly artifactEntitlements = toSignal(this.artifactEntitlementsService.list(), {initialValue: []});

  protected readonly targetName = computed(
    () => this.customerOrganizations().find((it) => it.id === this.targetCustomerOrganizationId)?.name
  );

  protected readonly sources = computed<License[]>(() =>
    this.customerOrganizations()
      .filter((customer) => customer.id !== this.targetCustomerOrganizationId)
      .map((customer) => ({
        customerOrganization: customer,
        licenseKeys: this.licenseKeys().filter((it) => it.customerOrganizationId === customer.id),
        applicationEntitlements: this.applicationEntitlements().filter(
          (it) => it.customerOrganizationId === customer.id
        ),
        artifactEntitlements: this.artifactEntitlements().filter((it) => it.customerOrganizationId === customer.id),
      }))
      .filter(
        (license) =>
          license.licenseKeys.length > 0 ||
          license.applicationEntitlements.length > 0 ||
          license.artifactEntitlements.length > 0
      )
  );

  private readonly inOneYear = dayjs().add(1, 'year').startOf('day').format('YYYY-MM-DD');
  protected readonly form = new FormGroup({
    sourceCustomerOrgId: new FormControl<string | null>(null, Validators.required),
    expiresAt: new FormControl(this.inOneYear, {nonNullable: true}),
  });

  private readonly sourceId = toSignal(this.form.controls.sourceCustomerOrgId.valueChanges, {
    initialValue: this.form.controls.sourceCustomerOrgId.value,
  });
  protected readonly selectedSource = computed(() => {
    const id = this.sourceId();
    return id ? this.sources().find((l) => l.customerOrganization.id === id) : undefined;
  });

  private readonly expiryOverrideIds = signal<ReadonlySet<string>>(new Set());
  private readonly expiresAt = toSignal(this.form.controls.expiresAt.valueChanges, {
    initialValue: this.form.controls.expiresAt.value,
  });
  protected readonly isNever = computed(() => !this.expiresAt());

  // License keys cannot be set to "never", so they are not selectable while no date is chosen.
  private readonly selectableOverrideIds = computed<string[]>(() => {
    const source = this.selectedSource();
    if (!source) {
      return [];
    }
    return [
      ...source.applicationEntitlements,
      ...source.artifactEntitlements,
      ...(this.isNever() ? [] : source.licenseKeys),
    ]
      .map((item) => item.id)
      .filter((id): id is string => id !== undefined);
  });

  protected readonly effectiveOverrideIds = computed<ReadonlySet<string>>(() => {
    const selectable = new Set(this.selectableOverrideIds());
    return new Set([...this.expiryOverrideIds()].filter((id) => selectable.has(id)));
  });

  protected readonly allOverrideSelected = computed(() => {
    const selectable = this.selectableOverrideIds();
    return selectable.length > 0 && selectable.every((id) => this.effectiveOverrideIds().has(id));
  });

  constructor() {
    super();
    Promise.all([
      this.customerOrganizationsService.refresh(),
      this.licenseKeysService.refresh(),
      this.applicationEntitlementsService.refresh(),
      this.artifactEntitlementsService.refresh(),
    ])
      .then(() => this.loading.set(false))
      .catch((e) => {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
        this.dialogRef.dismiss();
      });

    // Prefill the expiry with the latest expiration date among the selected source's licenses and
    // select every license for the override by default whenever the source changes.
    effect(() => {
      const source = this.selectedSource();
      this.form.controls.expiresAt.setValue(this.largestExpiry(source));
      this.expiryOverrideIds.set(
        new Set(
          [
            ...(source?.licenseKeys ?? []),
            ...(source?.applicationEntitlements ?? []),
            ...(source?.artifactEntitlements ?? []),
          ]
            .map((item) => item.id)
            .filter((id): id is string => id !== undefined)
        )
      );
    });
  }

  protected toggleOverride(id: string | undefined, checked: boolean): void {
    if (!id) {
      return;
    }
    this.expiryOverrideIds.update((ids) => {
      const next = new Set(ids);
      if (checked) {
        next.add(id);
      } else {
        next.delete(id);
      }
      return next;
    });
  }

  protected toggleAllOverrides(checked: boolean): void {
    this.expiryOverrideIds.set(checked ? new Set(this.selectableOverrideIds()) : new Set());
  }

  protected async copy() {
    this.form.markAllAsTouched();
    const source = this.selectedSource();
    if (!this.form.valid || !source) {
      return;
    }

    const targetId = this.targetCustomerOrganizationId;
    const overrideIds = this.effectiveOverrideIds();
    const isOverridden = (id: string | undefined) => id !== undefined && overrideIds.has(id);
    const expiresAt = this.form.controls.expiresAt.value;
    const entitlementExpiresAt = expiresAt ? new Date(expiresAt) : undefined;
    const licenseKeyExpiresAt = expiresAt ? dayjs(expiresAt).toISOString() : undefined;
    const creates = [
      ...source.artifactEntitlements.map((ae) =>
        this.artifactEntitlementsService.create({
          ...ae,
          id: undefined,
          customerOrganizationId: targetId,
          ...(isOverridden(ae.id) ? {expiresAt: entitlementExpiresAt} : {}),
        })
      ),
      ...source.applicationEntitlements.map((ae) =>
        this.applicationEntitlementsService.create({
          ...ae,
          id: undefined,
          customerOrganizationId: targetId,
          ...(isOverridden(ae.id) ? {expiresAt: entitlementExpiresAt} : {}),
        })
      ),
      ...source.licenseKeys.map(({id, name, ...lk}) =>
        this.licenseKeysService.create({
          ...lk,
          name: name!,
          customerOrganizationId: targetId,
          ...(isOverridden(id) ? {expiresAt: licenseKeyExpiresAt} : {}),
        })
      ),
    ];

    this.copying.set(true);
    try {
      await firstValueFrom(forkJoin(creates));
      this.toast.success('Licenses have been copied');
      await this.dialogRef.close(true);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.copying.set(false);
    }
  }

  private largestExpiry(source: License | undefined): string {
    const dates = [
      ...(source?.licenseKeys ?? []).map((lk) => lk.expiresAt),
      ...(source?.applicationEntitlements ?? []).map((ae) => ae.expiresAt),
      ...(source?.artifactEntitlements ?? []).map((ae) => ae.expiresAt),
    ].filter((value): value is string | Date => !!value);
    if (dates.length === 0) {
      return this.inOneYear;
    }
    const latest = dates.reduce((max, current) => (dayjs(current).isAfter(max) ? current : max));
    return dayjs(latest).format('YYYY-MM-DD');
  }
}
