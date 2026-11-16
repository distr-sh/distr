import {Component, input} from '@angular/core';
import {CustomDomainComponent} from '../components/custom-domains/custom-domain/custom-domain.component';
import {IdentityProvidersComponent} from '../components/custom-domains/identity-providers/identity-providers.component';

@Component({
  selector: 'app-customer-domain-and-sign-in',
  templateUrl: './customer-domain-and-sign-in.component.html',
  imports: [CustomDomainComponent, IdentityProvidersComponent],
  host: {class: 'block space-y-4'},
})
export class CustomerDomainAndSignInComponent {
  public readonly customerOrganizationId = input.required<string>();
}
