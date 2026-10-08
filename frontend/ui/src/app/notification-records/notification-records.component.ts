import {DatePipe, PercentPipe} from '@angular/common';
import {Component, computed, inject, input} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {switchMap} from 'rxjs';
import {NotificationRecordsService} from '../services/notification-records.service';
import {notificationCustomerNames} from './notification-record-display';

@Component({
  selector: 'app-notification-records',
  templateUrl: './notification-records.component.html',
  imports: [DatePipe, PercentPipe],
})
export class NotificationRecordsComponent {
  /** customerOrganizationId scopes the page to what one customer was notified about. */
  public readonly customerOrganizationId = input<string>();

  private readonly notificationRecordsService = inject(NotificationRecordsService);

  private readonly notificationRecords = toSignal(
    toObservable(this.customerOrganizationId).pipe(
      switchMap((customerOrganizationId) => this.notificationRecordsService.list(customerOrganizationId))
    )
  );

  protected readonly rows = computed(() =>
    (this.notificationRecords() ?? []).map((record) => {
      const customerNames = notificationCustomerNames(record);
      return {
        ...record,
        customerNames: customerNames.join(', '),
        customerNamesTitle: customerNames.join('\n'),
      };
    })
  );
}
