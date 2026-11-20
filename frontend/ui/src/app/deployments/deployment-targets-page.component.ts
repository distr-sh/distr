import {Component, inject} from '@angular/core';
import {refreshInBackground} from '../../util/refresh';
import {PageComponent} from '../components/page.component';
import {AuthService} from '../services/auth.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {DeploymentTargetsComponent} from './deployment-targets.component';

@Component({
  template: `
    <app-page>
      <app-deployment-targets />
    </app-page>
  `,
  imports: [PageComponent, DeploymentTargetsComponent],
})
export class DeploymentTargetsPageComponent {
  private readonly auth = inject(AuthService);
  private readonly customerOrganizations = inject(CustomerOrganizationsService);

  constructor() {
    if (this.auth.isVendor() || this.auth.isPartner()) {
      refreshInBackground(this.customerOrganizations.refresh());
    }
  }
}
