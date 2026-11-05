import {OverlayModule} from '@angular/cdk/overlay';
import {ChangeDetectionStrategy, Component, computed, ElementRef, inject, signal, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faBoxesStacked, faChevronDown} from '@fortawesome/free-solid-svg-icons';
import {refreshInBackground} from '../../util/refresh';
import {PageComponent} from '../components/page.component';
import {ApplicationEntitlementsService} from '../services/application-entitlements.service';
import {ArtifactEntitlementsService} from '../services/artifact-entitlements.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {LicenseKeysService} from '../services/license-keys.service';
import {ApplicationEntitlementsComponent} from './application-entitlements/application-entitlements.component';
import {ArtifactEntitlementsComponent} from './artifact-entitlements/artifact-entitlements.component';
import {LicenseKeysComponent} from './license-keys/license-keys.component';

@Component({
  selector: 'app-vendor-license-detail-page',
  imports: [
    FontAwesomeModule,
    OverlayModule,
    RouterLink,
    LicenseKeysComponent,
    ApplicationEntitlementsComponent,
    ArtifactEntitlementsComponent,
    PageComponent,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './vendor-license-detail-page.component.html',
})
export class VendorLicenseDetailPageComponent {
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faChevronDown = faChevronDown;

  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);

  protected readonly customerOrgId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );
  protected readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  protected readonly customer = computed(() => {
    const id = this.customerOrgId();
    return this.customerOrganizations()?.find((org) => org.id === id);
  });

  private readonly licenseKeys = toSignal(inject(LicenseKeysService).list());
  private readonly applicationEntitlements = toSignal(inject(ApplicationEntitlementsService).list());
  private readonly artifactEntitlements = toSignal(inject(ArtifactEntitlementsService).list());

  /** The first table with rows of this customer takes the "/" hotkey, the license keys when all are empty. */
  protected readonly hotkeyTable = computed(() => {
    const id = this.customerOrgId();
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

  protected readonly dropdownTriggerButton = viewChild.required<ElementRef<HTMLElement>>('dropdownTriggerButton');
  protected readonly breadcrumbDropdown = signal(false);
  breadcrumbDropdownWidth = 0;

  constructor() {
    refreshInBackground(this.customerOrganizationsService.refresh());
  }

  protected toggleBreadcrumbDropdown() {
    this.breadcrumbDropdown.update((v) => !v);
    if (this.breadcrumbDropdown()) {
      this.breadcrumbDropdownWidth = this.dropdownTriggerButton().nativeElement.getBoundingClientRect().width;
    }
  }
}
