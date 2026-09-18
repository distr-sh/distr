import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  ApplicationNotificationConfiguration,
  ArtifactNotificationConfiguration,
  CreateUpdateApplicationNotificationConfigurationRequest,
  CreateUpdateArtifactNotificationConfigurationRequest,
} from '../types/notification-configuration';

@Injectable({providedIn: 'root'})
export class ApplicationNotificationConfigurationsService {
  private readonly baseUrl = '/api/v1/application-notification-configurations';
  private readonly httpClient = inject(HttpClient);

  public list() {
    return this.httpClient.get<ApplicationNotificationConfiguration[]>(this.baseUrl);
  }

  public create(request: CreateUpdateApplicationNotificationConfigurationRequest) {
    return this.httpClient.post<ApplicationNotificationConfiguration>(this.baseUrl, request);
  }

  public update(id: string, request: CreateUpdateApplicationNotificationConfigurationRequest) {
    return this.httpClient.put<ApplicationNotificationConfiguration>(`${this.baseUrl}/${id}`, request);
  }

  public delete(id: string) {
    return this.httpClient.delete<void>(`${this.baseUrl}/${id}`);
  }
}

@Injectable({providedIn: 'root'})
export class ArtifactNotificationConfigurationsService {
  private readonly baseUrl = '/api/v1/artifact-notification-configurations';
  private readonly httpClient = inject(HttpClient);

  public list() {
    return this.httpClient.get<ArtifactNotificationConfiguration[]>(this.baseUrl);
  }

  public create(request: CreateUpdateArtifactNotificationConfigurationRequest) {
    return this.httpClient.post<ArtifactNotificationConfiguration>(this.baseUrl, request);
  }

  public update(id: string, request: CreateUpdateArtifactNotificationConfigurationRequest) {
    return this.httpClient.put<ArtifactNotificationConfiguration>(`${this.baseUrl}/${id}`, request);
  }

  public delete(id: string) {
    return this.httpClient.delete<void>(`${this.baseUrl}/${id}`);
  }
}
