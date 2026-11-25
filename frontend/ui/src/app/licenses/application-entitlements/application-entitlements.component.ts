import {DatePipe, NgPlural, NgPluralCase} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, input, signal, TemplateRef} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleExclamation, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, filter, first, firstValueFrom, switchMap} from 'rxjs';
import {isExpired} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {loadWhileOpen, refreshInBackground} from '../../../util/refresh';
import {AlertComponent} from '../../components/alert/alert.component';
import {SpinnerComponent} from '../../components/spinner/spinner.component';
import {ClickableRowDirective} from '../../directives/clickable-row.directive';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ApplicationsService} from '../../services/applications.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {ApplicationEntitlement} from '../../types/application-entitlement';
import {EditApplicationEntitlementComponent} from './edit-application-entitlement.component';

@Component({
  selector: 'app-application-entitlements',
  templateUrl: './application-entitlements.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    ReactiveFormsModule,
    FaIconComponent,
    DatePipe,
    NgPlural,
    NgPluralCase,
    EditApplicationEntitlementComponent,
    AlertComponent,
    ClickableRowDirective,
    SpinnerComponent,
  ],
})
export class ApplicationEntitlementsComponent {
  readonly customerOrganizationId = input<string>();
  readonly search = input.required<string>();

  protected readonly auth = inject(AuthService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly applicationsService = inject(ApplicationsService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  private readonly entitlements = toSignal(this.applicationEntitlementsService.list());
  protected readonly loaded = computed(() => this.entitlements() !== undefined);

  protected readonly filteredEntitlements = computed(() => {
    const search = this.search().toLowerCase();
    const customerOrganizationId = this.customerOrganizationId();
    return this.entitlements()?.filter(
      (it) =>
        (!customerOrganizationId || it.customerOrganizationId === customerOrganizationId) &&
        (!search || (it.name || '').toLowerCase().includes(search))
    );
  });

  applications$ = this.applicationsService.list();
  private readonly customers = toSignal(this.customerOrganizationsService.getCustomerOrganizations().pipe(first()));
  protected readonly customerName = computed(() => {
    const customerOrganizationId = this.customerOrganizationId();
    return customerOrganizationId ? this.customers()?.find((c) => c.id === customerOrganizationId)?.name : undefined;
  });

  editForm = new FormGroup({
    entitlement: new FormControl<ApplicationEntitlement | undefined>(undefined, {
      nonNullable: true,
    }),
  });

  editFormLoading = false;
  protected readonly drawerLoading = signal(false);

  private manageEntitlementDrawerRef?: DialogRef;

  constructor() {
    refreshInBackground(this.applicationEntitlementsService.refresh(), this.applicationsService.refresh());
  }

  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly isExpired = isExpired;

  async openDrawer(templateRef: TemplateRef<unknown>, entitlement?: ApplicationEntitlement) {
    this.hideDrawer();
    try {
      const loaded = await loadWhileOpen(
        this.drawerLoading,
        () => (this.manageEntitlementDrawerRef = this.overlay.showDrawer(templateRef)),
        Promise.all([
          entitlement?.id ? firstValueFrom(this.applicationEntitlementsService.get(entitlement.id)) : undefined,
          this.applicationsService.refresh(),
        ])
      );
      if (loaded === undefined) {
        return;
      }
      const [fresh] = loaded;
      if (fresh) {
        this.editForm.patchValue({entitlement: fresh});
      } else if (this.customerOrganizationId()) {
        this.editForm.patchValue({
          entitlement: {customerOrganizationId: this.customerOrganizationId()} as ApplicationEntitlement,
        });
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
