import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {Component, computed, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBuildingUser, faCopy, faKey, faMagnifyingGlass, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, forkJoin, startWith} from 'rxjs';
import {isExpired} from '../../util/dates';
import {getFormDisplayedError} from '../../util/errors';
import {SecureImagePipe} from '../../util/secureImage';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ApplicationEntitlementsService} from '../services/application-entitlements.service';
import {ArtifactEntitlementsService} from '../services/artifact-entitlements.service';
import {LicenseKeysService} from '../services/license-keys.service';
import {LicensesService} from '../services/licenses.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {License} from '../types/license';

@Component({
  selector: 'app-licenses-overview',
  imports: [AsyncPipe, ReactiveFormsModule, FaIconComponent, AutotrimDirective, SecureImagePipe],
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

  protected readonly copyForm = new FormGroup({
    sourceCustomerOrgId: new FormControl<string | null>(null, Validators.required),
  });

  protected readonly sourcesForCopy = computed(() => {
    const targetId = this.targetLicense()?.customerOrganization.id;
    return this.allLicenses().filter(
      (l) =>
        l.customerOrganization.id !== targetId &&
        (l.applicationEntitlements.length > 0 || l.artifactEntitlements.length > 0 || l.licenseKeys.length > 0)
    );
  });

  protected hasNoLicenses(license: License): boolean {
    return (
      license.applicationEntitlements.length === 0 &&
      license.artifactEntitlements.length === 0 &&
      license.licenseKeys.length === 0
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
    this.copyLicensesLoading.set(true);
    try {
      const creates = [
        ...source.artifactEntitlements.map((ae) =>
          this.artifactEntitlementsService.create({...ae, id: undefined, customerOrganizationId: targetId})
        ),
        ...source.applicationEntitlements.map((ae) =>
          this.applicationEntitlementsService.create({...ae, id: undefined, customerOrganizationId: targetId})
        ),
        ...source.licenseKeys.map((lk) =>
          this.licenseKeysService.create({...lk, id: undefined, token: '', customerOrganizationId: targetId})
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
