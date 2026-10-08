import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {AlertConfiguration, CreateUpdateAlertConfigurationRequest} from '../types/alert-configuration';
import {customerScopeParams} from './customer-scope';

const baseUrl = '/api/v1/alert-configurations';

@Injectable({providedIn: 'root'})
export class AlertConfigurationsService {
  private readonly httpClient = inject(HttpClient);

  public list(customerOrganizationId?: string) {
    return this.httpClient.get<AlertConfiguration[]>(baseUrl, {params: customerScopeParams(customerOrganizationId)});
  }

  public create(request: CreateUpdateAlertConfigurationRequest) {
    return this.httpClient.post<AlertConfiguration>(baseUrl, request);
  }

  public update(id: string, request: CreateUpdateAlertConfigurationRequest) {
    return this.httpClient.put<AlertConfiguration>(`${baseUrl}/${id}`, request);
  }

  public delete(id: string, customerOrganizationId?: string) {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`, {params: customerScopeParams(customerOrganizationId)});
  }
}
