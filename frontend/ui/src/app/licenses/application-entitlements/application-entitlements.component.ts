import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, inject, TemplateRef} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faCircleExclamation,
  faEye,
  faMagnifyingGlass,
  faPen,
  faPlus,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, filter, firstValueFrom, Observable, switchMap} from 'rxjs';
import {isExpired} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {filteredByFormControl} from '../../../util/filter';
import {drawerFlyInOut} from '../../animations/drawer';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ApplicationsService} from '../../services/applications.service';
import {AuthService} from '../../services/auth.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {ApplicationEntitlement} from '../../types/application-entitlement';
import {EditApplicationEntitlementComponent} from './edit-application-entitlement.component';

@Component({
  selector: 'app-application-entitlements',
  templateUrl: './application-entitlements.component.html',
  imports: [
    AsyncPipe,
    AutotrimDirective,
    ReactiveFormsModule,
    FaIconComponent,
    DatePipe,
    EditApplicationEntitlementComponent,
  ],
  animations: [dropdownAnimation, drawerFlyInOut, modalFlyInOut],
})
export class ApplicationEntitlementsComponent {
  protected readonly auth = inject(AuthService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  filterForm = new FormGroup({
    search: new FormControl(''),
  });

  licenses$: Observable<ApplicationEntitlement[]> = filteredByFormControl(
    this.applicationEntitlementsService.list(),
    this.filterForm.controls.search,
    (it: ApplicationEntitlement, search: string) =>
      !search || (it.name || '').toLowerCase().includes(search.toLowerCase())
  ).pipe(takeUntilDestroyed());

  applications$ = this.applicationsService.list();

  editForm = new FormGroup({
    license: new FormControl<ApplicationEntitlement | undefined>(undefined, {
      nonNullable: true,
      validators: Validators.required,
    }),
  });

  editFormLoading = false;

  private manageLicenseDrawerRef?: DialogRef;

  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faEye = faEye;
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly isExpired = isExpired;

  openDrawer(templateRef: TemplateRef<unknown>, license?: ApplicationEntitlement) {
    this.hideDrawer();
    if (license) {
      this.loadLicense(license);
    }
    this.manageLicenseDrawerRef = this.overlay.showDrawer(templateRef);
  }

  loadLicense(license: ApplicationEntitlement) {
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
      const action = license.id
        ? this.applicationEntitlementsService.update(license)
        : this.applicationEntitlementsService.create(license);
      try {
        const license = await firstValueFrom(action);
        this.hideDrawer();
        this.toast.success(`${license.name} saved successfully`);
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

  deleteLicense(license: ApplicationEntitlement) {
    this.overlay
      .confirm(`Really delete ${license.name}?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.applicationEntitlementsService.delete(license)),
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
}
