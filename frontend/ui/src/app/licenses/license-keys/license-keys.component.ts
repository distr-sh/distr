import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, inject, input, signal, TemplateRef, viewChild} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCopy, faEye, faMagnifyingGlass, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {catchError, combineLatest, EMPTY, filter, firstValueFrom, map, Observable, shareReplay, switchMap} from 'rxjs';
import {isExpired} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {filteredByFormControl} from '../../../util/filter';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {LicenseKeysService} from '../../services/license-keys.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {LicenseKey} from '../../types/license-key';
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
    AutotrimDirective,
  ],
  templateUrl: './license-keys.component.html',
})
export class LicenseKeysComponent {
  readonly customerOrganizationId = input<string>();

  protected readonly auth = inject(AuthService);
  private readonly licenseKeysService = inject(LicenseKeysService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly customerOrganizationService = inject(CustomerOrganizationsService);

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly faCopy = faCopy;
  protected readonly faEye = faEye;
  protected readonly isExpired = isExpired;

  protected readonly selectedLicense = signal<LicenseKey | undefined>(undefined);
  protected readonly selectedToken = signal<string | undefined>(undefined);
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
      validators: Validators.required,
    }),
  });
  editFormLoading = false;

  private manageLicenseDrawerRef?: DialogRef;

  private readonly customerOrganizations$ = this.customerOrganizationService
    .getCustomerOrganizations()
    .pipe(shareReplay(1));

  openDrawer(templateRef: TemplateRef<unknown>, license?: LicenseKey) {
    this.hideDrawer();
    if (license) {
      this.loadLicense(license);
    } else if (this.customerOrganizationId()) {
      this.editForm.patchValue({license: {customerOrganizationId: this.customerOrganizationId()} as LicenseKey});
    }
    this.manageLicenseDrawerRef = this.overlay.showDrawer(templateRef);
  }

  loadLicense(license: LicenseKey) {
    this.editForm.patchValue({license});
  }

  hideDrawer() {
    this.manageLicenseDrawerRef?.close();
    this.editForm.reset({license: undefined});
  }

  async saveLicense() {
    this.editForm.markAllAsTouched();
    const {license} = this.editForm.value;
    if (this.editForm.valid && license) {
      this.editFormLoading = true;
      const action = license.id ? this.licenseKeysService.update(license) : this.licenseKeysService.create(license);
      try {
        const saved = await firstValueFrom(action);
        this.hideDrawer();
        this.toast.success(`${saved.name} saved successfully`);
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.editFormLoading = false;
      }
    }
  }

  duplicateLicense(templateRef: TemplateRef<unknown>, license: LicenseKey) {
    this.openDrawer(templateRef, {
      ...license,
      id: undefined,
      name: '',
    });
  }

  deleteLicense(license: LicenseKey) {
    this.overlay
      .confirm(`Really delete ${license.name}? Note: deleting a license key does not revoke it.`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.licenseKeysService.delete(license)),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return EMPTY;
        })
      )
      .subscribe();
  }

  getOwnerColumn(customerOrganizationId?: string): Observable<string | undefined> {
    return customerOrganizationId
      ? this.customerOrganizations$.pipe(map((orgs) => orgs.find((o) => o.id === customerOrganizationId)?.name))
      : EMPTY;
  }

  viewLicense(license: LicenseKey) {
    this.viewLicenseLoading.set(true);
    this.licenseKeysService.getToken(license.id!).subscribe({
      next: ({token}) => {
        this.viewLicenseLoading.set(false);
        this.hideViewLicenseModal();
        this.selectedLicense.set(license);
        this.selectedToken.set(token);
        this.viewLicenseModalRef = this.overlay.showModal(this.viewLicenseModalTemplate(), {
          positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
        });
      },
      error: (e) => {
        this.viewLicenseLoading.set(false);
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      },
    });
  }

  hideViewLicenseModal() {
    this.viewLicenseModalRef?.close();
    this.selectedLicense.set(undefined);
    this.selectedToken.set(undefined);
  }
}
