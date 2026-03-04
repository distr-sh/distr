import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, inject, signal} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowLeft, faBuildingUser, faCopy, faEye, faPen, faPlus, faTrash} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, firstValueFrom, map, Observable, shareReplay, switchMap} from 'rxjs';
import {isExpired} from '../../util/dates';
import {SecureImagePipe} from '../../util/secureImage';
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
import {ViewLicenseKeyModalComponent} from './usage-licenses/view-usage-license-modal.component';

@Component({
  selector: 'app-customer-license-detail',
  imports: [AsyncPipe, DatePipe, FaIconComponent, SecureImagePipe, ViewLicenseKeyModalComponent],
  templateUrl: './customer-license-detail.component.html',
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
  protected readonly isExpired = isExpired;

  private viewLicenseKeyModalRef?: DialogRef<void>;
  protected selectedLicenseKey = signal<LicenseKey | undefined>(undefined);

  private readonly customerOrgId$ = this.route.paramMap.pipe(map((params) => params.get('customerOrganizationId')!));

  protected readonly customer$ = this.customerOrgId$.pipe(
    switchMap((id) => this.customerOrgsService.getCustomerOrganizationById(id)),
    shareReplay(1)
  );

  protected readonly appEntitlements$: Observable<ApplicationEntitlement[]> = this.customerOrgId$.pipe(
    switchMap(() =>
      this.appEntitlementsService.list().pipe(
        map((entitlements) =>
          entitlements.filter(
            (e) => e.customerOrganizationId === this.route.snapshot.paramMap.get('customerOrganizationId')
          )
        ),
        catchError(() => EMPTY)
      )
    ),
    shareReplay(1)
  );

  protected readonly artifactEntitlements$: Observable<ArtifactEntitlement[]> = this.customerOrgId$.pipe(
    switchMap(() =>
      this.artifactEntitlementsService.list().pipe(
        map((entitlements) =>
          entitlements.filter(
            (e) => e.customerOrganizationId === this.route.snapshot.paramMap.get('customerOrganizationId')
          )
        ),
        catchError(() => EMPTY)
      )
    ),
    shareReplay(1)
  );

  protected readonly licenseKeys$: Observable<LicenseKey[]> = this.customerOrgId$.pipe(
    switchMap(() =>
      this.licenseKeysService.list().pipe(
        map((keys) =>
          keys.filter((k) => k.customerOrganizationId === this.route.snapshot.paramMap.get('customerOrganizationId'))
        ),
        catchError(() => EMPTY)
      )
    ),
    shareReplay(1)
  );

  protected goBack() {
    this.router.navigate(['/licenses']);
  }

  protected async deleteAppEntitlement(entitlement: ApplicationEntitlement) {
    try {
      await firstValueFrom(this.appEntitlementsService.delete(entitlement));
      this.toast.success(`${entitlement.name} deleted`);
      this.appEntitlementsService.refresh();
    } catch {
      this.toast.error('Failed to delete entitlement');
    }
  }

  protected async deleteArtifactEntitlement(entitlement: ArtifactEntitlement) {
    try {
      await firstValueFrom(this.artifactEntitlementsService.delete(entitlement));
      this.toast.success(`${entitlement.name} deleted`);
      this.artifactEntitlementsService.refresh();
    } catch {
      this.toast.error('Failed to delete entitlement');
    }
  }

  protected async deleteLicenseKey(licenseKey: LicenseKey) {
    try {
      await firstValueFrom(this.licenseKeysService.delete(licenseKey));
      this.toast.success(`${licenseKey.name} deleted`);
      this.licenseKeysService.refresh();
    } catch {
      this.toast.error('Failed to delete license key');
    }
  }

  protected viewLicenseKey(licenseKey: LicenseKey) {
    this.hideViewLicenseKeyModal();
    this.selectedLicenseKey.set(licenseKey);
    this.viewLicenseKeyModalRef = this.overlay.showModal(ViewLicenseKeyModalComponent, {
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  private hideViewLicenseKeyModal() {
    this.viewLicenseKeyModalRef?.close();
    this.viewLicenseKeyModalRef = undefined;
  }
}
