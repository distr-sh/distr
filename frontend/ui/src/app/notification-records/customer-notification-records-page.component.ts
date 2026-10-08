import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {CustomerBreadcrumbComponent} from '../components/customer-breadcrumb.component';
import {PageComponent} from '../components/page.component';
import {NotificationRecordsComponent} from './notification-records.component';

@Component({
  template: `
    <app-page>
      <div class="distr-table-panel">
        <app-customer-breadcrumb [customerOrganizationId]="customerOrganizationId()" section="notification-history" />
        <app-notification-records [customerOrganizationId]="customerOrganizationId()" />
      </div>
    </app-page>
  `,
  imports: [PageComponent, CustomerBreadcrumbComponent, NotificationRecordsComponent],
})
export class CustomerNotificationRecordsPageComponent {
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );
}
