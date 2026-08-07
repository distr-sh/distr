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

  // See CustomDomainsService.list: returns everything within the caller's scope, filtered client-side.
  public list(): Observable<CustomOidcConfigurationsResponse> {
    return this.httpClient.get<CustomOidcConfigurationsResponse>(baseUrl);
  }

  // request.customerOrganizationId targets a customer's own provider instead of the caller's own.
  public create(request: CustomOidcConfigurationRequest): Observable<CustomOidcConfiguration> {
    return this.httpClient.post<CustomOidcConfiguration>(baseUrl, request);
  }

  public update(id: string, request: CustomOidcConfigurationRequest): Observable<CustomOidcConfiguration> {
    return this.httpClient.put<CustomOidcConfiguration>(`${baseUrl}/${id}`, request);
  }

  public delete(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }

  public test(id: string): Observable<void> {
    return this.httpClient.post<void>(`${baseUrl}/${id}/test`, {});
  }
}
