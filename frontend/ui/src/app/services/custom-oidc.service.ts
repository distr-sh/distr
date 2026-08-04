import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {
  CustomOidcConfiguration,
  CustomOidcConfigurationRequest,
  CustomOidcConfigurationsResponse,
} from '../types/custom-oidc';

const baseUrl = '/api/v1/custom-oidc';

@Injectable({
  providedIn: 'root',
})
export class CustomOidcService {
  private readonly httpClient = inject(HttpClient);

  public list(): Observable<CustomOidcConfigurationsResponse> {
    return this.httpClient.get<CustomOidcConfigurationsResponse>(baseUrl);
  }

  public create(request: CustomOidcConfigurationRequest): Observable<CustomOidcConfiguration> {
    return this.httpClient.post<CustomOidcConfiguration>(baseUrl, request);
  }

  public update(id: string, request: CustomOidcConfigurationRequest): Observable<CustomOidcConfiguration> {
    return this.httpClient.put<CustomOidcConfiguration>(`${baseUrl}/${id}`, request);
  }

  public delete(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }

  /** Runs discovery against the stored issuer and reports the provider's own error verbatim. */
  public test(id: string): Observable<void> {
    return this.httpClient.post<void>(`${baseUrl}/${id}/test`, {});
  }
}
