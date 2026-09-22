import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateUpdateNotificationConfigurationRequest,
  UpdateNotificationConfiguration,
} from '../types/notification-configuration';
import {customerScopeParams} from './customer-scope';

@Injectable({providedIn: 'root'})
export class UpdateNotificationConfigurationsService {
  private readonly baseUrl = '/api/v1/update-notification-configurations';
  private readonly httpClient = inject(HttpClient);

  public list(customerOrganizationId?: string) {
    return this.httpClient.get<UpdateNotificationConfiguration[]>(this.baseUrl, {
      params: customerScopeParams(customerOrganizationId),
    });
  }

  public create(request: CreateUpdateNotificationConfigurationRequest) {
    return this.httpClient.post<UpdateNotificationConfiguration>(this.baseUrl, request);
  }

  public update(id: string, request: CreateUpdateNotificationConfigurationRequest) {
    return this.httpClient.put<UpdateNotificationConfiguration>(`${this.baseUrl}/${id}`, request);
  }

  public delete(id: string, customerOrganizationId?: string) {
    return this.httpClient.delete<void>(`${this.baseUrl}/${id}`, {
      params: customerScopeParams(customerOrganizationId),
    });
  }
}
