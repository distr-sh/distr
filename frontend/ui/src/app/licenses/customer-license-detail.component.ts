import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArrowLeft,
  faBuildingUser,
  faCopy,
  faEye,
  faPen,
  faPlus,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, filter, firstValueFrom, map, switchMap} from 'rxjs';
import {isExpired} from '../../util/dates';
import {getFormDisplayedError} from '../../util/errors';
import {SecureImagePipe} from '../../util/secureImage';
import {drawerFlyInOut} from '../animations/drawer';
import {ApplicationEntitlementsService} from '../services/application-entitlements.service';
import {ArtifactEntitlementsService} from '../services/artifact-entitlements.service';
import {AuthService} from '../services/auth.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {LicenseKeysService} from '../services/license-keys.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {ApplicationEntitlement} from '../types/application-entitlement';
import {ArtifactEntitlement} from '../types/artifact-entitlement';
import {LicenseKey} from '../types/license-key';
import {EditApplicationEntitlementComponent} from './application-entitlements/edit-application-entitlement.component';
import {EditArtifactEntitlementComponent} from './artifact-entitlements/edit-artifact-entitlement.component';
import {EditLicenseKeyComponent} from './license-keys/edit-license-key.component';
import {ViewLicenseKeyModalComponent} from './license-keys/view-license-key-modal.component';

@Component({
  selector: 'app-customer-license-detail',
  imports: [
    AsyncPipe,
    DatePipe,
    FaIconComponent,
    ReactiveFormsModule,
    SecureImagePipe,
    EditApplicationEntitlementComponent,
    EditArtifactEntitlementComponent,
    EditLicenseKeyComponent,
    ViewLicenseKeyModalComponent,
  ],
  templateUrl: './customer-license-detail.component.html',
  animations: [drawerFlyInOut],
})
export class CustomerLicenseDetailComponent {
  protected readonly auth = inject(AuthService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly customerOrgsService = inject(CustomerOrganizationsService);
  private readonly appEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly artifactEntitlementsService = inject(ArtifactEntitlementsService);
  private readonly licenseKeysService = inject(LicenseKeysService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  protected readonly faArrowLeft = faArrowLeft;
  protected readonly faBuildingUser = faBuildingUser;
  protected readonly faPlus = faPlus;
  protected readonly faPen = faPen;
  protected readonly faTrash = faTrash;
  protected readonly faCopy = faCopy;
  protected readonly faEye = faEye;
  protected readonly faXmark = faXmark;
  protected readonly isExpired = isExpired;

  // View license key modal
  private readonly viewLicenseModalTemplate = viewChild.required<TemplateRef<unknown>>('viewLicenseKeyModal');
  private viewLicenseKeyModalRef?: DialogRef;
  protected selectedLicenseKey = signal<LicenseKey | undefined>(undefined);

  // Drawer
  private drawerRef?: DialogRef;

  // Edit forms
  licenseKeyEditForm = new FormGroup({
    license: new FormControl<LicenseKey | undefined>(undefined, {
      nonNullable: true,
      validators: Validators.required,
    }),
  });

  appEntitlementEditForm = new FormGroup({
    license: new FormControl<ApplicationEntitlement | undefined>(undefined, {
      nonNullable: true,
      validators: Validators.required,
    }),
  });

  artifactEntitlementEditForm = new FormGroup({
    license: new FormControl<ArtifactEntitlement | undefined>(undefined, {
      nonNullable: true,
      validators: Validators.required,
    }),
  });

  editFormLoading = false;

  // Data as signals
  private readonly customerOrgId = toSignal(
    this.route.paramMap.pipe(map((params) => params.get('customerOrganizationId')!))
  );

  protected readonly customer = toSignal(
    this.route.paramMap.pipe(
      map((params) => params.get('customerOrganizationId')!),
      switchMap((id) => this.customerOrgsService.getCustomerOrganizationById(id))
    )
  );

  protected readonly appEntitlements = toSignal(
    this.route.paramMap.pipe(
      map((params) => params.get('customerOrganizationId')!),
      switchMap((id) =>
        this.appEntitlementsService
          .list()
          .pipe(map((entitlements) => entitlements.filter((e) => e.customerOrganizationId === id)))
      ),
      catchError(() => EMPTY)
    ),
    {initialValue: []}
  );

  protected readonly artifactEntitlements = toSignal(
    this.route.paramMap.pipe(
      map((params) => params.get('customerOrganizationId')!),
      switchMap((id) =>
        this.artifactEntitlementsService
          .list()
          .pipe(map((entitlements) => entitlements.filter((e) => e.customerOrganizationId === id)))
      ),
      catchError(() => EMPTY)
    ),
    {initialValue: []}
  );

  protected readonly licenseKeys = toSignal(
    this.route.paramMap.pipe(
      map((params) => params.get('customerOrganizationId')!),
      switchMap((id) =>
        this.licenseKeysService.list().pipe(map((keys) => keys.filter((k) => k.customerOrganizationId === id)))
      ),
      catchError(() => EMPTY)
    ),
    {initialValue: []}
  );

  protected goBack() {
    this.router.navigate(['/licenses']);
  }

  // Drawer management
  openDrawer(templateRef: TemplateRef<unknown>) {
    this.hideDrawer();
    this.drawerRef = this.overlay.showDrawer(templateRef);
  }

  hideDrawer() {
    this.drawerRef?.close();
    this.licenseKeyEditForm.reset({license: undefined});
    this.appEntitlementEditForm.reset({license: undefined});
    this.artifactEntitlementEditForm.reset({license: undefined});
  }

  // License Key CRUD
  openLicenseKeyDrawer(templateRef: TemplateRef<unknown>, licenseKey?: LicenseKey) {
    this.licenseKeyEditForm.reset({license: undefined});
    if (licenseKey) {
      this.licenseKeyEditForm.patchValue({license: licenseKey});
    } else {
      this.licenseKeyEditForm.patchValue({
        license: {customerOrganizationId: this.customerOrgId()} as LicenseKey,
      });
    }
    this.openDrawer(templateRef);
  }

  duplicateLicenseKey(templateRef: TemplateRef<unknown>, licenseKey: LicenseKey) {
    this.openLicenseKeyDrawer(templateRef, {
      ...licenseKey,
      id: undefined,
      name: '',
      token: '',
    });
  }

  async saveLicenseKey() {
    this.licenseKeyEditForm.markAllAsTouched();
    const {license} = this.licenseKeyEditForm.value;
    if (this.licenseKeyEditForm.valid && license) {
      this.editFormLoading = true;
      const action = license.id ? this.licenseKeysService.update(license) : this.licenseKeysService.create(license);
      try {
        const saved = await firstValueFrom(action);
        this.hideDrawer();
        this.toast.success(`${saved.name} saved successfully`);
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) this.toast.error(msg);
      } finally {
        this.editFormLoading = false;
      }
    }
  }

  deleteLicenseKey(licenseKey: LicenseKey) {
    this.overlay
      .confirm(`Really delete ${licenseKey.name}?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.licenseKeysService.delete(licenseKey)),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) this.toast.error(msg);
          return EMPTY;
        })
      )
      .subscribe();
  }

  viewLicenseKey(licenseKey: LicenseKey) {
    this.hideViewLicenseKeyModal();
    this.selectedLicenseKey.set(licenseKey);
    this.viewLicenseKeyModalRef = this.overlay.showModal(this.viewLicenseModalTemplate(), {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  hideViewLicenseKeyModal() {
    this.viewLicenseKeyModalRef?.close();
    this.selectedLicenseKey.set(undefined);
  }

  // Application Entitlement CRUD
  openAppEntitlementDrawer(templateRef: TemplateRef<unknown>, entitlement?: ApplicationEntitlement) {
    this.appEntitlementEditForm.reset({license: undefined});
    if (entitlement) {
      this.appEntitlementEditForm.patchValue({license: entitlement});
    } else {
      this.appEntitlementEditForm.patchValue({
        license: {customerOrganizationId: this.customerOrgId()} as ApplicationEntitlement,
      });
    }
    this.openDrawer(templateRef);
  }

  async saveAppEntitlement() {
    this.appEntitlementEditForm.markAllAsTouched();
    const {license} = this.appEntitlementEditForm.value;
    if (this.appEntitlementEditForm.valid && license) {
      this.editFormLoading = true;
      const action = license.id
        ? this.appEntitlementsService.update(license)
        : this.appEntitlementsService.create(license);
      try {
        const saved = await firstValueFrom(action);
        this.hideDrawer();
        this.toast.success(`${saved.name} saved successfully`);
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) this.toast.error(msg);
      } finally {
        this.editFormLoading = false;
      }
    }
  }

  deleteAppEntitlement(entitlement: ApplicationEntitlement) {
    this.overlay
      .confirm(`Really delete ${entitlement.name}?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.appEntitlementsService.delete(entitlement)),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) this.toast.error(msg);
          return EMPTY;
        })
      )
      .subscribe();
  }

  // Artifact Entitlement CRUD
  openArtifactEntitlementDrawer(templateRef: TemplateRef<unknown>, entitlement?: ArtifactEntitlement) {
    this.artifactEntitlementEditForm.reset({license: undefined});
    if (entitlement) {
      this.artifactEntitlementEditForm.patchValue({license: entitlement});
    } else {
      this.artifactEntitlementEditForm.patchValue({
        license: {customerOrganizationId: this.customerOrgId()} as ArtifactEntitlement,
      });
    }
    this.openDrawer(templateRef);
  }

  async saveArtifactEntitlement() {
    this.artifactEntitlementEditForm.markAllAsTouched();
    const {license} = this.artifactEntitlementEditForm.value;
    if (this.artifactEntitlementEditForm.valid && license) {
      this.editFormLoading = true;
      const action = license.id
        ? this.artifactEntitlementsService.update(license)
        : this.artifactEntitlementsService.create(license);
      try {
        const saved = await firstValueFrom(action);
        this.hideDrawer();
        this.toast.success(`${saved.name} saved successfully`);
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) this.toast.error(msg);
      } finally {
        this.editFormLoading = false;
      }
    }
  }

  deleteArtifactEntitlement(entitlement: ArtifactEntitlement) {
    this.overlay
      .confirm(`Really delete ${entitlement.name}?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.artifactEntitlementsService.delete(entitlement)),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) this.toast.error(msg);
          return EMPTY;
        })
      )
      .subscribe();
  }
}
