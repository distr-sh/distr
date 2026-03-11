import {OverlayModule} from '@angular/cdk/overlay';
import {Component, computed, ElementRef, inject, signal, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faBoxesStacked, faChevronDown} from '@fortawesome/free-solid-svg-icons';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {ApplicationEntitlementsComponent} from './application-entitlements/application-entitlements.component';
import {ArtifactEntitlementsComponent} from './artifact-entitlements/artifact-entitlements.component';
import {LicenseKeysComponent} from './license-keys/license-keys.component';

@Component({
  selector: 'app-customer-license-detail',
  imports: [
    FontAwesomeModule,
    OverlayModule,
    RouterLink,
    LicenseKeysComponent,
    ApplicationEntitlementsComponent,
    ArtifactEntitlementsComponent,
  ],
  templateUrl: './customer-license-detail.component.html',
})
export class CustomerLicenseDetailComponent {
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

  protected readonly dropdownTriggerButton = viewChild.required<ElementRef<HTMLElement>>('dropdownTriggerButton');
  protected readonly breadcrumbDropdown = signal(false);
  breadcrumbDropdownWidth = 0;

  protected toggleBreadcrumbDropdown() {
    this.breadcrumbDropdown.update((v) => !v);
    if (this.breadcrumbDropdown()) {
      this.breadcrumbDropdownWidth = this.dropdownTriggerButton().nativeElement.getBoundingClientRect().width;
    }
  }
}
