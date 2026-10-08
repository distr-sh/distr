import {Component} from '@angular/core';
import {faClockRotateLeft} from '@fortawesome/free-solid-svg-icons';
import {OrganizationScopeNavComponent} from '../components/organization-scope-nav.component';
import {PageComponent} from '../components/page.component';
import {NotificationRecordsComponent} from './notification-records.component';

@Component({
  template: `
    <app-page>
      <div class="distr-table-panel">
        <app-organization-scope-nav
          [icon]="faClockRotateLeft"
          label="Organization History"
          linkLabel="Show customer history"
          section="notification-history" />
        <app-notification-records />
      </div>
    </app-page>
  `,
  imports: [PageComponent, OrganizationScopeNavComponent, NotificationRecordsComponent],
})
export class NotificationRecordsPageComponent {
  protected readonly faClockRotateLeft = faClockRotateLeft;
}
