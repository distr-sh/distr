import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {AlertConfiguration, CreateUpdateAlertConfigurationRequest} from '../types/alert-configuration';

const baseUrl = '/api/v1/alert-configurations';

@Injectable({providedIn: 'root'})
export class AlertConfigurationsService {
  private readonly httpClient = inject(HttpClient);

  public list() {
    return this.httpClient.get<AlertConfiguration[]>(baseUrl);
  }

  public create(request: CreateUpdateAlertConfigurationRequest) {
    return this.httpClient.post<AlertConfiguration>(baseUrl, request);
  }

  public update(id: string, request: CreateUpdateAlertConfigurationRequest) {
    return this.httpClient.put<AlertConfiguration>(`${baseUrl}/${id}`, request);
  }

  public delete(id: string) {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }
}
