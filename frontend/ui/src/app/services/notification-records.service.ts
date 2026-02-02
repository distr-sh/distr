import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {NotificationRecord} from '../types/notification-record';

@Injectable({providedIn: 'root'})
export class NotificationRecordsService {
  private readonly httpClient = inject(HttpClient);

  public list() {
    return this.httpClient.get<NotificationRecord[]>('/api/v1/notification-records');
  }
}
