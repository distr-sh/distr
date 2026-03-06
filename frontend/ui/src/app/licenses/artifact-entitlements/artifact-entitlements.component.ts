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
import {catchError, combineLatest, EMPTY, filter, firstValueFrom, map, Observable, shareReplay, switchMap} from 'rxjs';
import {isExpired} from '../../../util/dates';
import {getFormDisplayedError} from '../../../util/errors';
import {filteredByFormControl} from '../../../util/filter';
import {drawerFlyInOut} from '../../animations/drawer';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {ArtifactEntitlementsService} from '../../services/artifact-entitlements.service';
import {ArtifactsService} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {ArtifactEntitlement, ArtifactEntitlementSelection} from '../../types/artifact-entitlement';
import {EditArtifactEntitlementComponent} from './edit-artifact-entitlement.component';

@Component({
  selector: 'app-artifact-entitlements',
  imports: [ReactiveFormsModule, AsyncPipe, FaIconComponent, DatePipe, EditArtifactEntitlementComponent],
  templateUrl: './artifact-entitlements.component.html',
  animations: [dropdownAnimation, drawerFlyInOut, modalFlyInOut],
})
export class ArtifactEntitlementsComponent {
  readonly customerOrganizationId = input<string>();

  protected readonly auth = inject(AuthService);
  private readonly artifactEntitlementsService = inject(ArtifactEntitlementsService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly customerOrganizationService = inject(CustomerOrganizationsService);
  private readonly artifactsService = inject(ArtifactsService);

  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faEye = faEye;
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly isExpired = isExpired;

  filterForm = new FormGroup({
    search: new FormControl(''),
  });

  licenses$: Observable<ArtifactEntitlement[]> = combineLatest([
    filteredByFormControl(
      this.artifactEntitlementsService.list(),
      this.filterForm.controls.search,
      (it: ArtifactEntitlement, search: string) =>
        !search || (it.name || '').toLowerCase().includes(search.toLowerCase())
    ),
    toObservable(this.customerOrganizationId),
  ]).pipe(
    map(([entitlements, id]) => (id ? entitlements.filter((e) => e.customerOrganizationId === id) : entitlements)),
    takeUntilDestroyed()
  );

  editForm = new FormGroup({
    license: new FormControl<ArtifactEntitlement | undefined>(undefined, {
      nonNullable: true,
      validators: Validators.required,
    }),
  });
  editFormLoading = false;

  private manageLicenseDrawerRef?: DialogRef;

  private readonly customerOrganizations$ = this.customerOrganizationService
    .getCustomerOrganizations()
    .pipe(shareReplay(1));
  private readonly artifacts$ = this.artifactsService.list();

  openDrawer(templateRef: TemplateRef<unknown>, license?: ArtifactEntitlement) {
    this.hideDrawer();
    if (license) {
      this.loadLicense(license);
    } else if (this.customerOrganizationId()) {
      this.editForm.patchValue({
        license: {customerOrganizationId: this.customerOrganizationId()} as ArtifactEntitlement,
      });
    }
    this.manageLicenseDrawerRef = this.overlay.showDrawer(templateRef);
  }

  loadLicense(license: ArtifactEntitlement) {
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
        ? this.artifactEntitlementsService.update(license)
        : this.artifactEntitlementsService.create(license);
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

  deleteLicense(license: ArtifactEntitlement) {
    this.overlay
      .confirm(`Really delete ${license.name}?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.artifactEntitlementsService.delete(license)),
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

  getArtifactColumn(selection?: ArtifactEntitlementSelection[]): Observable<string | undefined> {
    return selection?.[0]?.artifactId
      ? this.artifacts$.pipe(
          map((artifacts) => artifacts.find((a) => a.id === selection?.[0]?.artifactId)),
          map((a) => a?.name + (selection?.length > 1 ? ' (+' + (selection.length - 1) + ')' : ''))
        )
      : EMPTY;
  }

  getOwnerColumn(customerOrganizationId?: string): Observable<string | undefined> {
    return customerOrganizationId
      ? this.customerOrganizations$.pipe(map((users) => users.find((u) => u.id === customerOrganizationId)?.name))
      : EMPTY;
  }
}
