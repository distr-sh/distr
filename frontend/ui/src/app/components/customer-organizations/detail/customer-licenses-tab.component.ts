import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCopy} from '@fortawesome/free-solid-svg-icons';
import {EMPTY} from 'rxjs';
import {ApplicationEntitlementsComponent} from '../../../licenses/application-entitlements/application-entitlements.component';
import {ArtifactEntitlementsComponent} from '../../../licenses/artifact-entitlements/artifact-entitlements.component';
import {CopyLicensesContext, CopyLicensesModalComponent} from '../../../licenses/copy-licenses-modal.component';
import {LicenseKeysComponent} from '../../../licenses/license-keys/license-keys.component';
import {ApplicationEntitlementsService} from '../../../services/application-entitlements.service';
import {ArtifactEntitlementsService} from '../../../services/artifact-entitlements.service';
import {AuthService} from '../../../services/auth.service';
import {LicenseKeysService} from '../../../services/license-keys.service';
import {OverlayService} from '../../../services/overlay.service';
import {AlertComponent} from '../../alert/alert.component';

@Component({
  templateUrl: './customer-licenses-tab.component.html',
  host: {class: 'flex flex-col gap-6'},
  imports: [
    AlertComponent,
    FaIconComponent,
    LicenseKeysComponent,
    ApplicationEntitlementsComponent,
    ArtifactEntitlementsComponent,
  ],
})
export class CustomerLicensesTabComponent {
  private readonly overlay = inject(OverlayService);

  protected readonly faCopy = faCopy;
  private readonly canEdit = inject(AuthService).hasAnyRole('read_write', 'admin');

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );

  private readonly licenseKeys = toSignal(inject(LicenseKeysService).list());
  private readonly applicationEntitlements = toSignal(inject(ApplicationEntitlementsService).list());
  private readonly artifactEntitlements = toSignal(inject(ArtifactEntitlementsService).list());

  /** The first table with rows of this customer takes the "/" hotkey, the license keys when all are empty. */
  protected readonly hotkeyTable = computed(() => {
    const id = this.customerOrganizationId();
    const hasRows = (items: {customerOrganizationId?: string}[] | undefined) =>
      items?.some((it) => it.customerOrganizationId === id) ?? false;
    if (hasRows(this.licenseKeys())) {
      return 'licenseKeys';
    } else if (hasRows(this.applicationEntitlements())) {
      return 'applicationEntitlements';
    } else if (hasRows(this.artifactEntitlements())) {
      return 'artifactEntitlements';
    } else {
      return 'licenseKeys';
    }
  });

  /** Offers the copy only to a customer without any licenses, once there is another customer to copy from. */
  protected readonly canCopyFromAnotherCustomer = computed(() => {
    const id = this.customerOrganizationId();
    const licenseKeys = this.licenseKeys();
    const applicationEntitlements = this.applicationEntitlements();
    const artifactEntitlements = this.artifactEntitlements();
    if (!this.canEdit || !licenseKeys || !applicationEntitlements || !artifactEntitlements) {
      return false;
    }
    const all = [...licenseKeys, ...applicationEntitlements, ...artifactEntitlements];
    return (
      !all.some((it) => it.customerOrganizationId === id) &&
      all.some((it) => it.customerOrganizationId !== undefined && it.customerOrganizationId !== id)
    );
  });

  protected openCopyModal() {
    const targetCustomerOrganizationId = this.customerOrganizationId();
    if (targetCustomerOrganizationId) {
      const data: CopyLicensesContext = {targetCustomerOrganizationId};
      this.overlay.showModal<boolean>(CopyLicensesModalComponent, {data});
    }
  }
}
