import {Component} from '@angular/core';
import {CustomOidcComponent} from '../../organization-settings/custom-oidc/custom-oidc.component';

@Component({
  selector: 'app-customer-portal-domain',
  templateUrl: './customer-portal-domain.component.html',
  imports: [CustomOidcComponent],
})
export class CustomerPortalDomainComponent {}
