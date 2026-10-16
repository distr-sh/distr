import {OverlayModule} from '@angular/cdk/overlay';
import {Component, computed, ElementRef, inject, input, signal, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBoxesStacked, faChevronDown} from '@fortawesome/free-solid-svg-icons';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';

/** The breadcrumb of a page that shows one customer's rows, with the dropdown to switch customer. */
@Component({
  selector: 'app-customer-breadcrumb',
  templateUrl: './customer-breadcrumb.component.html',
  imports: [RouterLink, FaIconComponent, OverlayModule],
})
export class CustomerBreadcrumbComponent {
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faChevronDown = faChevronDown;

  public readonly customerOrganizationId = input.required<string | undefined>();
  /** section is the route below the customer that the dropdown keeps the user on. */
  public readonly section = input.required<string>();

  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  protected readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.customerOrganizationId())
  );

  private readonly dropdownTriggerButton = viewChild.required<ElementRef<HTMLElement>>('dropdownTriggerButton');
  protected readonly dropdownOpen = signal(false);
  protected readonly dropdownWidth = signal(0);

  protected toggleDropdown() {
    this.dropdownOpen.update((open) => !open);
    if (this.dropdownOpen()) {
      this.dropdownWidth.set(this.dropdownTriggerButton().nativeElement.getBoundingClientRect().width);
    }
  }
}
