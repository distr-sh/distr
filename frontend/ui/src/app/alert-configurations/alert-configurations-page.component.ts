import {Component} from '@angular/core';
import {faBell} from '@fortawesome/free-solid-svg-icons';
import {OrganizationScopeNavComponent} from '../components/organization-scope-nav.component';
import {PageComponent} from '../components/page.component';
import {AlertConfigurationsComponent} from './alert-configurations.component';

@Component({
  template: `
    <app-page>
      <div class="distr-table-panel">
        <app-organization-scope-nav
          [icon]="faBell"
          label="Organization Alerts"
          linkLabel="Show customer alerts"
          section="alerts" />
        <app-alert-configurations />
      </div>
    </app-page>
  `,
  imports: [PageComponent, OrganizationScopeNavComponent, AlertConfigurationsComponent],
})
export class AlertConfigurationsPageComponent {
  protected readonly faBell = faBell;
}
