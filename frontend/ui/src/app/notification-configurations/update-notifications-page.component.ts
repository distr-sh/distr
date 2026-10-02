import {Component} from '@angular/core';
import {faCircleArrowUp} from '@fortawesome/free-solid-svg-icons';
import {OrganizationScopeNavComponent} from '../components/organization-scope-nav.component';
import {PageComponent} from '../components/page.component';
import {UpdateNotificationsComponent} from './update-notifications.component';

@Component({
  template: `
    <app-page>
      <div class="distr-table-panel">
        <app-organization-scope-nav
          [icon]="faCircleArrowUp"
          label="Organization Updates"
          linkLabel="Show customer updates"
          section="updates" />
        <app-update-notifications />
      </div>
    </app-page>
  `,
  imports: [PageComponent, OrganizationScopeNavComponent, UpdateNotificationsComponent],
})
export class UpdateNotificationsPageComponent {
  protected readonly faCircleArrowUp = faCircleArrowUp;
}
