import {OverlayModule} from '@angular/cdk/overlay';
import {ChangeDetectionStrategy, Component, computed, ElementRef, inject, signal, viewChild} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faBoxesStacked, faChevronDown} from '@fortawesome/free-solid-svg-icons';
import {map, of, switchMap} from 'rxjs';
import {CustomOidcComponent} from '../organization-settings/custom-oidc.component';
import {ContextService} from '../services/context.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';

@Component({
  selector: 'app-customer-settings',
  templateUrl: './customer-settings.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [CustomOidcComponent, RouterLink, FontAwesomeModule, OverlayModule],
})
export class CustomerSettingsComponent {
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faChevronDown = faChevronDown;

  private readonly route = inject(ActivatedRoute);
  private readonly contextService = inject(ContextService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);

  // The vendor reaches the page through the customer list and gets the id from the route, a customer
  // admin reaches its own page and gets it from the authenticated context.
  private readonly routeCustomerOrganizationId = toSignal(
    this.route.paramMap.pipe(map((params) => params.get('customerOrganizationId') ?? undefined)),
    {initialValue: undefined}
  );
  private readonly contextCustomerOrganization = toSignal(this.contextService.getCustomerOrganization(), {
    initialValue: undefined,
  });

  // The customer switcher only makes sense for a vendor, who has more than one customer to switch
  // between; a customer admin only ever sees their own organization here.
  protected readonly vendorScoped = computed(() => !!this.routeCustomerOrganizationId());

  protected readonly customerOrganizationId = computed(
    () => this.routeCustomerOrganizationId() ?? this.contextCustomerOrganization()?.id
  );

  protected readonly customerOrganizations = toSignal(
    toObservable(this.vendorScoped).pipe(
      switchMap((vendorScoped) =>
        vendorScoped ? this.customerOrganizationsService.getCustomerOrganizations() : of([])
      )
    )
  );
  protected readonly customerOrganization = computed(() => {
    const id = this.customerOrganizationId();
    return this.customerOrganizations()?.find((org) => org.id === id);
  });

  protected readonly dropdownTriggerButton = viewChild<ElementRef<HTMLElement>>('dropdownTriggerButton');
  protected readonly breadcrumbDropdown = signal(false);
  breadcrumbDropdownWidth = 0;

  protected toggleBreadcrumbDropdown() {
    this.breadcrumbDropdown.update((v) => !v);
    if (this.breadcrumbDropdown()) {
      this.breadcrumbDropdownWidth = this.dropdownTriggerButton()?.nativeElement.getBoundingClientRect().width ?? 0;
    }
  }
}
