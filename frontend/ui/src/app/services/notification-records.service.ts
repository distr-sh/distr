import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {NotificationRecord} from '../types/notification-record';
import {customerScopeParams} from './customer-scope';

@Injectable({providedIn: 'root'})
export class NotificationRecordsService {
  private readonly httpClient = inject(HttpClient);

  public list(customerOrganizationId?: string) {
    return this.httpClient.get<NotificationRecord[]>('/api/v1/notification-records', {
      params: customerScopeParams(customerOrganizationId),
    });
  }
}
