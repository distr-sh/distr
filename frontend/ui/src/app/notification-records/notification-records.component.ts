import {DatePipe} from '@angular/common';
import {Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {PageComponent} from '../components/page.component';
import {NotificationRecordsService} from '../services/notification-records.service';

@Component({
  templateUrl: './notification-records.component.html',
  imports: [DatePipe, PageComponent],
})
export class NotificationRecordsComponent {
  private readonly notificationRecordsService = inject(NotificationRecordsService);

  protected readonly notificationRecords = toSignal(this.notificationRecordsService.list());
}
