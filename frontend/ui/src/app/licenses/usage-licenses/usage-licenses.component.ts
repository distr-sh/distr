import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCopy, faEye, faMagnifyingGlass, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, filter, firstValueFrom, map, Observable, shareReplay, switchMap} from 'rxjs';
import {isExpired} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {filteredByFormControl} from '../../../util/filter';
import {drawerFlyInOut} from '../../animations/drawer';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {UsageLicensesService} from '../../services/usage-licenses.service';
import {UsageLicense} from '../../types/usage-license';
import {EditUsageLicenseComponent} from './edit-usage-license.component';
import {ViewUsageLicenseModalComponent} from './view-usage-license-modal.component';

@Component({
  selector: 'app-usage-licenses',
  imports: [
    ReactiveFormsModule,
    AsyncPipe,
    DatePipe,
    FaIconComponent,
    EditUsageLicenseComponent,
    ViewUsageLicenseModalComponent,
    AutotrimDirective,
  ],
  templateUrl: './usage-licenses.component.html',
  animations: [drawerFlyInOut],
})
export class UsageLicensesComponent {
  protected readonly auth = inject(AuthService);
  private readonly usageLicensesService = inject(UsageLicensesService);
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

  protected readonly selectedLicense = signal<UsageLicense | undefined>(undefined);
  private readonly viewLicenseModalTemplate = viewChild.required<TemplateRef<unknown>>('viewLicenseModal');
  private viewLicenseModalRef?: DialogRef;

  filterForm = new FormGroup({
    search: new FormControl(''),
  });

  licenses$: Observable<UsageLicense[]> = filteredByFormControl(
    this.usageLicensesService.list(),
    this.filterForm.controls.search,
    (it: UsageLicense, search: string) => !search || (it.name || '').toLowerCase().includes(search.toLowerCase())
  ).pipe(takeUntilDestroyed());

  editForm = new FormGroup({
    license: new FormControl<UsageLicense | undefined>(undefined, {
      nonNullable: true,
      validators: Validators.required,
    }),
  });
  editFormLoading = false;

  private manageLicenseDrawerRef?: DialogRef;

  private readonly customerOrganizations$ = this.customerOrganizationService
    .getCustomerOrganizations()
    .pipe(shareReplay(1));

  openDrawer(templateRef: TemplateRef<unknown>, license?: UsageLicense) {
    this.hideDrawer();
    if (license) {
      this.loadLicense(license);
    }
    this.manageLicenseDrawerRef = this.overlay.showDrawer(templateRef);
  }

  loadLicense(license: UsageLicense) {
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
      const action = license.id ? this.usageLicensesService.update(license) : this.usageLicensesService.create(license);
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

  duplicateLicense(templateRef: TemplateRef<unknown>, license: UsageLicense) {
    this.openDrawer(templateRef, {
      ...license,
      id: undefined,
      name: '',
      token: '',
    });
  }

  deleteLicense(license: UsageLicense) {
    this.overlay
      .confirm(`Really delete ${license.name}?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.usageLicensesService.delete(license)),
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

  viewLicense(license: UsageLicense) {
    this.hideViewLicenseModal();
    this.selectedLicense.set(license);
    this.viewLicenseModalRef = this.overlay.showModal(this.viewLicenseModalTemplate(), {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  hideViewLicenseModal() {
    this.viewLicenseModalRef?.close();
    this.selectedLicense.set(undefined);
  }
}
