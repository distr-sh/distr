import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, inject, input, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toObservable} from '@angular/core/rxjs-interop';
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
import {catchError, combineLatest, EMPTY, filter, firstValueFrom, map, Observable, switchMap} from 'rxjs';
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
  readonly customerOrganizationId = input<string>();

  protected readonly auth = inject(AuthService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  filterForm = new FormGroup({
    search: new FormControl(''),
  });

  entitlements$: Observable<ApplicationEntitlement[]> = combineLatest([
    filteredByFormControl(
      this.applicationEntitlementsService.list(),
      this.filterForm.controls.search,
      (it: ApplicationEntitlement, search: string) =>
        !search || (it.name || '').toLowerCase().includes(search.toLowerCase())
    ),
    toObservable(this.customerOrganizationId),
  ]).pipe(
    map(([entitlements, id]) => (id ? entitlements.filter((e) => e.customerOrganizationId === id) : entitlements)),
    takeUntilDestroyed()
  );

  applications$ = this.applicationsService.list();

  editForm = new FormGroup({
    entitlement: new FormControl<ApplicationEntitlement | undefined>(undefined, {
      nonNullable: true,
      validators: Validators.required,
    }),
  });

  editFormLoading = false;

  private manageEntitlementDrawerRef?: DialogRef;

  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faEye = faEye;
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly isExpired = isExpired;

  openDrawer(templateRef: TemplateRef<unknown>, entitlement?: ApplicationEntitlement) {
    this.hideDrawer();
    if (entitlement) {
      this.loadEntitlement(entitlement);
    } else if (this.customerOrganizationId()) {
      this.editForm.patchValue({
        entitlement: {customerOrganizationId: this.customerOrganizationId()} as ApplicationEntitlement,
      });
    }
    this.manageEntitlementDrawerRef = this.overlay.showDrawer(templateRef);
  }

  loadEntitlement(entitlement: ApplicationEntitlement) {
    this.editForm.patchValue({entitlement});
  }

  hideDrawer() {
    this.manageEntitlementDrawerRef?.close();
    this.editForm.reset({entitlement: undefined});
  }

  async saveEntitlement() {
    this.editForm.markAllAsTouched();
    const {entitlement} = this.editForm.value;
    if (this.editForm.valid && entitlement) {
      this.editFormLoading = true;
      const action = entitlement.id
        ? this.applicationEntitlementsService.update(entitlement)
        : this.applicationEntitlementsService.create(entitlement);
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

  deleteEntitlement(entitlement: ApplicationEntitlement) {
    this.overlay
      .confirm(`Really delete ${entitlement.name}?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.applicationEntitlementsService.delete(entitlement)),
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
