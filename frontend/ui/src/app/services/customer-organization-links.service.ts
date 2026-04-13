import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {CustomerOrganizationLink} from '@distr-sh/distr-sdk';
import {Observable} from 'rxjs';

@Injectable({providedIn: 'root'})
export class CustomerOrganizationLinksService {
  private readonly httpClient = inject(HttpClient);

  private baseUrl(customerOrganizationId: string): string {
    return `/api/v1/customer-organizations/${customerOrganizationId}/links`;
  }

  public list(customerOrganizationId: string): Observable<CustomerOrganizationLink[]> {
    return this.httpClient.get<CustomerOrganizationLink[]>(this.baseUrl(customerOrganizationId));
  }

  public create(customerOrganizationId: string, name: string, link: string): Observable<CustomerOrganizationLink> {
    return this.httpClient.post<CustomerOrganizationLink>(this.baseUrl(customerOrganizationId), {name, link});
  }

  public update(
    customerOrganizationId: string,
    linkId: string,
    name: string,
    link: string
  ): Observable<CustomerOrganizationLink> {
    return this.httpClient.put<CustomerOrganizationLink>(`${this.baseUrl(customerOrganizationId)}/${linkId}`, {
      name,
      link,
    });
  }

  public delete(customerOrganizationId: string, linkId: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.baseUrl(customerOrganizationId)}/${linkId}`);
  }
}
