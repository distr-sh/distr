import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {ApplicationEntitlementsComponent} from '../../../licenses/application-entitlements/application-entitlements.component';
import {ArtifactEntitlementsComponent} from '../../../licenses/artifact-entitlements/artifact-entitlements.component';
import {LicenseKeysComponent} from '../../../licenses/license-keys/license-keys.component';
import {ApplicationEntitlementsService} from '../../../services/application-entitlements.service';
import {ArtifactEntitlementsService} from '../../../services/artifact-entitlements.service';
import {LicenseKeysService} from '../../../services/license-keys.service';

@Component({
  templateUrl: './customer-licenses-tab.component.html',
  host: {class: 'flex flex-col gap-6'},
  imports: [LicenseKeysComponent, ApplicationEntitlementsComponent, ArtifactEntitlementsComponent],
})
export class CustomerLicensesTabComponent {
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
}
