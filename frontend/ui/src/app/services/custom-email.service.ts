import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {
  CustomEmailConfiguration,
  CustomEmailSettings,
  UpdateCustomEmailConfigurationRequest,
} from '../types/custom-email';

const baseUrl = '/api/v1/custom-email';

@Injectable({
  providedIn: 'root',
})
export class CustomEmailService {
  private readonly httpClient = inject(HttpClient);

  /** Responds with 404 when the organization has no email configuration. */
  public get(): Observable<CustomEmailConfiguration> {
    return this.httpClient.get<CustomEmailConfiguration>(baseUrl);
  }

  public upsert(request: UpdateCustomEmailConfigurationRequest): Observable<CustomEmailConfiguration> {
    return this.httpClient.put<CustomEmailConfiguration>(baseUrl, request);
  }

  public delete(): Observable<void> {
    return this.httpClient.delete<void>(baseUrl);
  }

  public test(settings: CustomEmailSettings): Observable<void> {
    return this.httpClient.post<void>(`${baseUrl}/test`, settings);
  }
}
