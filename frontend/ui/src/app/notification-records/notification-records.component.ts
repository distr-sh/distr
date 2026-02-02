import {DatePipe} from '@angular/common';
import {Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {StatusDotComponent} from '../components/status-dot';
import {NotificationRecordsService} from '../services/notification-records.service';

@Component({
  templateUrl: './notification-records.component.html',
  imports: [DatePipe, StatusDotComponent],
})
export class NotificationRecordsComponent {
  private readonly notificationRecordsService = inject(NotificationRecordsService);

  protected readonly notificationRecords = toSignal(this.notificationRecordsService.list());
}
