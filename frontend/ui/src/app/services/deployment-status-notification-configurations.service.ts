import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateUpdateDeploymentStatusNotificationConfigurationRequest,
  DeploymentStatusNotificationConfiguration,
} from '../types/deployment-status-notification-configuration';

const baseUrl = '/api/v1/deployment-status-notification-configurations';

@Injectable({providedIn: 'root'})
export class DeploymentStatusNotificationConfigurationsService {
  private readonly httpClient = inject(HttpClient);

  public list() {
    return this.httpClient.get<DeploymentStatusNotificationConfiguration[]>(baseUrl);
  }

  public create(request: CreateUpdateDeploymentStatusNotificationConfigurationRequest) {
    return this.httpClient.post<DeploymentStatusNotificationConfiguration>(baseUrl, request);
  }

  public update(id: string, request: CreateUpdateDeploymentStatusNotificationConfigurationRequest) {
    return this.httpClient.put<DeploymentStatusNotificationConfiguration>(`${baseUrl}/${id}`, request);
  }

  public delete(id: string) {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }
}
