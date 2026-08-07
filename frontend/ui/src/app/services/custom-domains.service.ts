import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {CreateCustomDomainRequest, CustomDomain} from '../types/custom-domain';

const baseUrl = '/api/v1/custom-domains';

@Injectable({
  providedIn: 'root',
})
export class CustomDomainsService {
  private readonly httpClient = inject(HttpClient);

  // Returns every domain within the caller's scope: the vendor's own and every customer's for a
  // vendor, one customer's own for a customer. Callers filter the response down to the section they
  // render, the same way SecretsPage filters an unscoped list rather than asking the server for a slice.
  public list(): Observable<CustomDomain[]> {
    return this.httpClient.get<CustomDomain[]>(baseUrl);
  }

  // customerOrganizationId targets a customer's own domain instead of the caller's own; only a vendor
  // or partner admin may set it, a customer may only ever target itself.
  public create(requests: CreateCustomDomainRequest[], customerOrganizationId?: string): Observable<CustomDomain[]> {
    return this.httpClient.post<CustomDomain[]>(baseUrl, {domains: requests, customerOrganizationId});
  }

  public delete(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }

  public verify(id: string): Observable<CustomDomain> {
    return this.httpClient.post<CustomDomain>(`${baseUrl}/${id}/verify`, {});
  }
}
