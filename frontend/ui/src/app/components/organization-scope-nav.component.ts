import {Component, computed, inject, input} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowRight, IconDefinition} from '@fortawesome/free-solid-svg-icons';
import {of} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';

/** The header of a page that shows the organization's own rows, linking to the customers' rows. */
@Component({
  selector: 'app-organization-scope-nav',
  templateUrl: './organization-scope-nav.component.html',
  imports: [RouterLink, FaIconComponent],
})
export class OrganizationScopeNavComponent {
  protected readonly faArrowRight = faArrowRight;

  public readonly icon = input.required<IconDefinition>();
  public readonly label = input.required<string>();
  public readonly linkLabel = input.required<string>();
  /** section is the route below the customer that the link leads to. */
  public readonly section = input.required<string>();

  private readonly auth = inject(AuthService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly customerOrganizations = toSignal(
    this.auth.isVendor() || this.auth.isPartner()
      ? this.customerOrganizationsService.getCustomerOrganizations()
      : of([])
  );
  protected readonly firstCustomerOrganization = computed(() => this.customerOrganizations()?.[0]);
}
