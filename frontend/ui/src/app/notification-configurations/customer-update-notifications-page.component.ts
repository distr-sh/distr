import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {CustomerBreadcrumbComponent} from '../components/customer-breadcrumb.component';
import {PageComponent} from '../components/page.component';
import {UpdateNotificationsComponent} from './update-notifications.component';

@Component({
  template: `
    <app-page>
      <div class="distr-table-panel">
        <app-customer-breadcrumb [customerOrganizationId]="customerOrganizationId()" section="updates" />
        <app-update-notifications [customerOrganizationId]="customerOrganizationId()" />
      </div>
    </app-page>
  `,
  imports: [PageComponent, CustomerBreadcrumbComponent, UpdateNotificationsComponent],
})
export class CustomerUpdateNotificationsPageComponent {
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );
}
