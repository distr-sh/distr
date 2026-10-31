import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {ChangeDetectionStrategy, Component, inject, input, signal, TemplateRef, viewChild} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleExclamation, faCopy, faEye, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {catchError, combineLatest, EMPTY, filter, firstValueFrom, map, Observable, switchMap} from 'rxjs';
import {isExpired} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {filteredByFormControl} from '../../../util/filter';
import {loadWhileOpen, refreshInBackground} from '../../../util/refresh';
import {AlertComponent} from '../../components/alert/alert.component';
import {ClipComponent} from '../../components/clip.component';
import {SearchBarComponent} from '../../components/search-bar.component';
import {SpinnerComponent} from '../../components/spinner/spinner.component';
import {UuidComponent} from '../../components/uuid';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {LicenseKeysService} from '../../services/license-keys.service';
import {LicenseTemplatesService} from '../../services/license-templates.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {AffectedDeployment} from '../../types/affected-deployment';
import {LicenseKey, UpdateLicenseKeyRequest} from '../../types/license-key';
import {EditLicenseKeyComponent} from './edit-license-key.component';
import {ViewLicenseKeyModalComponent} from './view-license-key-modal.component';

@Component({
  selector: 'app-license-keys',
  imports: [
    ReactiveFormsModule,
    AsyncPipe,
    DatePipe,
    FaIconComponent,
    EditLicenseKeyComponent,
    ViewLicenseKeyModalComponent,
    UuidComponent,
    ClipComponent,
    SearchBarComponent,
    AlertComponent,
    SpinnerComponent,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './license-keys.component.html',
})
export class LicenseKeysComponent {
  readonly customerOrganizationId = input<string>();

  protected readonly auth = inject(AuthService);
  protected readonly features = inject(FeatureFlagService);
  private readonly licenseKeysService = inject(LicenseKeysService);
  private readonly licenseTemplatesService = inject(LicenseTemplatesService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly customerOrganizationService = inject(CustomerOrganizationsService);

  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly faCopy = faCopy;
  protected readonly faEye = faEye;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly isExpired = isExpired;

  protected readonly selectedLicense = signal<LicenseKey | undefined>(undefined);
  protected readonly selectedToken = signal<string | undefined>(undefined);
  protected readonly affectedDeployments = signal<AffectedDeployment[]>([]);
  protected readonly viewLicenseLoading = signal(false);
  private readonly viewLicenseModalTemplate = viewChild.required<TemplateRef<unknown>>('viewLicenseModal');
  private viewLicenseModalRef?: DialogRef;

  filterForm = new FormGroup({
    search: new FormControl(''),
  });

  protected readonly filteredLicenses = toSignal(
    combineLatest([
      filteredByFormControl(
        this.licenseKeysService.list(),
        this.filterForm.controls.search,
        (it: LicenseKey, search: string) => !search || (it.name || '').toLowerCase().includes(search.toLowerCase())
      ),
      toObservable(this.customerOrganizationId),
    ]).pipe(
      map(([keys, id]) => (id ? keys.filter((k) => k.customerOrganizationId === id) : keys)),
      takeUntilDestroyed()
    ),
    {initialValue: [] as LicenseKey[]}
  );

  editForm = new FormGroup({
    license: new FormControl<LicenseKey | undefined>(undefined, {
      nonNullable: true,
    }),
  });
  editFormLoading = false;

  protected readonly drawerLoading = signal(false);

  constructor() {
    refreshInBackground(this.licenseKeysService.refresh());
    this.editForm.valueChanges.pipe(takeUntilDestroyed()).subscribe(() => this.affectedDeployments.set([]));
  }

  private manageLicenseDrawerRef?: DialogRef;

  private readonly customerOrganizations$ = this.customerOrganizationService.getCustomerOrganizations();

  openDrawer(templateRef: TemplateRef<unknown>, license?: LicenseKey) {
    return this.showDrawer(templateRef, license?.id, (fresh) => fresh);
  }

  duplicateLicense(templateRef: TemplateRef<unknown>, license: LicenseKey) {
    return this.showDrawer(templateRef, license.id, (fresh) => ({...fresh, id: undefined, name: ''}));
  }

  private async showDrawer(
    templateRef: TemplateRef<unknown>,
    licenseId: string | undefined,
    prepare: (license: LicenseKey) => LicenseKey
  ) {
    this.hideDrawer();
    try {
      const loaded = await loadWhileOpen(
        this.drawerLoading,
        () => (this.manageLicenseDrawerRef = this.overlay.showDrawer(templateRef)),
        Promise.all([
          this.licenseKeysService.refresh(),
          this.features.isVendorBillingEnabled() ? this.licenseTemplatesService.refresh() : [],
        ])
      );
      if (loaded === undefined) {
        return;
      }
      if (licenseId) {
        const license = loaded[0].find((it) => it.id === licenseId);
        if (!license) {
          this.toast.error('This license key no longer exists');
          this.hideDrawer();
          return;
        }
        this.editForm.patchValue({license: prepare(license)});
      } else if (this.customerOrganizationId()) {
        this.editForm.patchValue({license: {customerOrganizationId: this.customerOrganizationId()} as LicenseKey});
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
      this.hideDrawer();
    }
  }

  hideDrawer() {
    this.manageLicenseDrawerRef?.close();
    this.editForm.reset({license: undefined});
    this.affectedDeployments.set([]);
  }

  async saveLicense() {
    this.editForm.markAllAsTouched();
    const {license} = this.editForm.value;
    if (this.editForm.valid && license) {
      this.editFormLoading = true;
      try {
        const licenseKeyFields = {
          description: license.description,
          payload: license.payload,
          notBefore: license.notBefore,
          expiresAt: license.expiresAt,
          licenseTemplateId: license.licenseTemplateId,
        };
        const saved = license.id
          ? await this.updateLicense(license.id, licenseKeyFields)
          : await firstValueFrom(
              this.licenseKeysService.create({
                ...licenseKeyFields,
                name: license.name!,
                customerOrganizationId: license.customerOrganizationId,
              })
            );
        this.hideDrawer();
        if (!license.id && saved.licenseTemplateId) {
          this.toast.success(`${saved.name} saved successfully. Link the license key ID in your Stripe dashboard.`);
        } else {
          this.toast.success(`${saved.name} saved successfully`);
        }
      } catch (e) {
        if (e instanceof HttpErrorResponse && e.status === 409) {
          this.affectedDeployments.set(e.error.affectedDeployments);
          return;
        }
        this.affectedDeployments.set([]);
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.editFormLoading = false;
      }
    }
  }

  private updateLicense(id: string, request: UpdateLicenseKeyRequest): Promise<LicenseKey> {
    const confirm = this.affectedDeployments().length > 0;
    return firstValueFrom(this.licenseKeysService.update(id, request, confirm));
  }

  deleteLicense(license: LicenseKey) {
    this.overlay
      .confirm(`Really delete ${license.name}? Note: deleting a license key does not revoke it.`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.licenseKeysService.delete(license)),
        catchError((e) => {
          if (e instanceof HttpErrorResponse && e.status === 409) {
            const affected: AffectedDeployment[] = e.error?.affectedDeployments ?? [];
            if (affected.length > 0) {
              const names = affected.map((a) => a.deploymentTargetName).join(', ');
              this.toast.error(
                `Cannot delete: this license key is referenced by ${names}. Remove the reference from the deployment config first.`
              );
              return EMPTY;
            }
          }

          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return EMPTY;
        })
      )
      .subscribe();
  }

  protected getLicenseKeyReference(name: string): string {
    return /^[a-zA-Z_]\w*$/.test(name) ? `{{ .LicenseKeys.${name} }}` : `{{ index .LicenseKeys "${name}" }}`;
  }

  getOwnerColumn(customerOrganizationId?: string): Observable<string | undefined> {
    return customerOrganizationId
      ? this.customerOrganizations$.pipe(map((orgs) => orgs.find((o) => o.id === customerOrganizationId)?.name))
      : EMPTY;
  }

  async viewLicense(license: LicenseKey) {
    this.viewLicenseLoading.set(true);
    try {
      const [{token}, licenses] = await Promise.all([
        firstValueFrom(this.licenseKeysService.getToken(license.id!)),
        this.licenseKeysService.refresh(),
      ]);
      this.hideViewLicenseModal();
      this.selectedLicense.set(licenses.find((it) => it.id === license.id) ?? license);
      this.selectedToken.set(token);
      this.viewLicenseModalRef = this.overlay.showModal(this.viewLicenseModalTemplate(), {
        positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
      });
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.viewLicenseLoading.set(false);
    }
  }

  hideViewLicenseModal() {
    this.viewLicenseModalRef?.close();
    this.selectedLicense.set(undefined);
    this.selectedToken.set(undefined);
  }
}
