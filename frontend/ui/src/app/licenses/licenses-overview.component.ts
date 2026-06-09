import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe} from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  signal,
  TemplateRef,
  viewChild,
} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBuildingUser, faCopy, faKey, faMagnifyingGlass, faXmark} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {firstValueFrom, forkJoin, startWith} from 'rxjs';
import {isExpired} from '../../util/dates';
import {getFormDisplayedError} from '../../util/errors';
import {SecureImagePipe} from '../../util/secureImage';
import {ExpiresAtPickerComponent} from '../components/expires-at-picker/expires-at-picker.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ApplicationEntitlementsService} from '../services/application-entitlements.service';
import {ArtifactEntitlementsService} from '../services/artifact-entitlements.service';
import {AuthService} from '../services/auth.service';
import {LicenseKeysService} from '../services/license-keys.service';
import {LicensesService} from '../services/licenses.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {License} from '../types/license';

@Component({
  selector: 'app-licenses-overview',
  imports: [
    AsyncPipe,
    DatePipe,
    ReactiveFormsModule,
    FaIconComponent,
    AutotrimDirective,
    SecureImagePipe,
    ExpiresAtPickerComponent,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './licenses-overview.component.html',
})
export class LicensesOverviewComponent {
  private readonly licensesService = inject(LicensesService);
  private readonly router = inject(Router);
  private readonly overlay = inject(OverlayService);
  private readonly artifactEntitlementsService = inject(ArtifactEntitlementsService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly licenseKeysService = inject(LicenseKeysService);
  private readonly toast = inject(ToastService);
  protected readonly auth = inject(AuthService);

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faBuildingUser = faBuildingUser;
  protected readonly faKey = faKey;
  protected readonly faCopy = faCopy;
  protected readonly faXmark = faXmark;

  protected readonly filterForm = new FormGroup({
    search: new FormControl(''),
  });

  private readonly allLicenses = toSignal(this.licensesService.list(), {initialValue: []});

  private readonly filterValue = toSignal(
    this.filterForm.controls.search.valueChanges.pipe(startWith(this.filterForm.controls.search.value))
  );

  protected readonly licenses = computed(() => {
    const search = this.filterValue()?.toLowerCase();
    const all = this.allLicenses();
    return !search ? all : all.filter((l) => l.customerOrganization.name.toLowerCase().includes(search));
  });

  private readonly copyLicensesModalTemplate = viewChild.required<TemplateRef<unknown>>('copyLicensesModal');
  private copyLicensesModalRef?: DialogRef;
  protected readonly targetLicense = signal<License | undefined>(undefined);
  protected readonly copyLicensesLoading = signal(false);

  private readonly inOneYear = dayjs().add(1, 'year').startOf('day').format('YYYY-MM-DD');
  protected readonly copyForm = new FormGroup({
    sourceCustomerOrgId: new FormControl<string | null>(null, Validators.required),
    expiresAt: new FormControl(this.inOneYear, {nonNullable: true}),
  });

  private readonly copySourceId = toSignal(this.copyForm.controls.sourceCustomerOrgId.valueChanges, {
    initialValue: this.copyForm.controls.sourceCustomerOrgId.value,
  });
  protected readonly selectedCopySource = computed(() => {
    const id = this.copySourceId();
    return id ? this.allLicenses().find((l) => l.customerOrganization.id === id) : undefined;
  });

  protected readonly expiryOverrideIds = signal<ReadonlySet<string>>(new Set());
  private readonly copyExpiry = toSignal(this.copyForm.controls.expiresAt.valueChanges, {
    initialValue: this.copyForm.controls.expiresAt.value,
  });
  protected readonly isNever = computed(() => !this.copyExpiry());

  // License keys cannot be set to "never", so they are not selectable while no date is chosen.
  protected readonly selectableOverrideIds = computed<string[]>(() => {
    const source = this.selectedCopySource();
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

  protected readonly sourcesForCopy = computed(() => {
    const targetId = this.targetLicense()?.customerOrganization.id;
    return this.allLicenses().filter((l) => l.customerOrganization.id !== targetId && !this.hasNoLicenses(l));
  });

  constructor() {
    // Prefill the expiry with the latest expiration date among the selected source's licenses and
    // select every license for the override by default whenever the source changes.
    effect(() => {
      const source = this.selectedCopySource();
      this.copyForm.controls.expiresAt.setValue(this.largestExpiry(source));
      this.expiryOverrideIds.set(new Set(this.allLicenseIds(source)));
    });
  }

  private allLicenseIds(source: License | undefined): string[] {
    if (!source) {
      return [];
    }
    return [...source.licenseKeys, ...source.applicationEntitlements, ...source.artifactEntitlements]
      .map((item) => item.id)
      .filter((id): id is string => id !== undefined);
  }

  protected isOverrideSelected(id: string | undefined): boolean {
    return !!id && this.effectiveOverrideIds().has(id);
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

  protected hasNoLicenses(license: License): boolean {
    return (
      license.applicationEntitlements.length === 0 &&
      license.artifactEntitlements.length === 0 &&
      license.licenseKeys.length === 0
    );
  }

  protected canCopyFromAnotherCustomer(license: License): boolean {
    return this.allLicenses().some(
      (l) => l.customerOrganization.id !== license.customerOrganization.id && !this.hasNoLicenses(l)
    );
  }

  protected navigateToCustomer(license: License) {
    this.router.navigate(['/licenses', license.customerOrganization.id]);
  }

  protected countExpired(license: License): number {
    let count = 0;
    for (const ae of license.applicationEntitlements) {
      if (isExpired(ae)) count++;
    }
    for (const ae of license.artifactEntitlements) {
      if (isExpired(ae)) count++;
    }
    for (const lk of license.licenseKeys) {
      if (isExpired(lk)) count++;
    }
    return count;
  }

  protected openCopyModal(event: Event, license: License) {
    event.stopPropagation();
    this.targetLicense.set(license);
    this.copyForm.reset();
    this.copyLicensesModalRef = this.overlay.showModal(this.copyLicensesModalTemplate(), {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  protected closeCopyModal() {
    this.copyLicensesModalRef?.dismiss();
    this.targetLicense.set(undefined);
  }

  protected async copyLicenses() {
    this.copyForm.markAllAsTouched();
    const sourceOrgId = this.copyForm.controls.sourceCustomerOrgId.value;
    const target = this.targetLicense();
    if (!this.copyForm.valid || !sourceOrgId || !target) {
      return;
    }
    const source = this.allLicenses().find((l) => l.customerOrganization.id === sourceOrgId);
    if (!source) return;

    const targetId = target.customerOrganization.id;
    const expiresAt = this.copyForm.controls.expiresAt.value;
    const entitlementExpiresAt = expiresAt ? new Date(expiresAt) : undefined;
    const licenseKeyExpiresAt = expiresAt ? dayjs(expiresAt).toISOString() : undefined;
    this.copyLicensesLoading.set(true);
    try {
      const creates = [
        ...source.artifactEntitlements.map((ae) =>
          this.artifactEntitlementsService.create({
            ...ae,
            id: undefined,
            customerOrganizationId: targetId,
            ...(this.isOverrideSelected(ae.id) ? {expiresAt: entitlementExpiresAt} : {}),
          })
        ),
        ...source.applicationEntitlements.map((ae) =>
          this.applicationEntitlementsService.create({
            ...ae,
            id: undefined,
            customerOrganizationId: targetId,
            ...(this.isOverrideSelected(ae.id) ? {expiresAt: entitlementExpiresAt} : {}),
          })
        ),
        ...source.licenseKeys.map(({id: _, name, ...lk}) =>
          this.licenseKeysService.create({
            ...lk,
            name: name!,
            customerOrganizationId: targetId,
            ...(this.isOverrideSelected(_) ? {expiresAt: licenseKeyExpiresAt} : {}),
          })
        ),
      ];
      if (creates.length > 0) {
        await firstValueFrom(forkJoin(creates));
      }
      this.closeCopyModal();
      this.router.navigate(['/licenses', targetId]);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.copyLicensesLoading.set(false);
    }
  }
}
