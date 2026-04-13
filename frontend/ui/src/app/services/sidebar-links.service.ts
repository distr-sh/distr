import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {SidebarLink} from '@distr-sh/distr-sdk';
import {Observable} from 'rxjs';

@Injectable({providedIn: 'root'})
export class SidebarLinksService {
  private readonly httpClient = inject(HttpClient);

  private baseUrl(customerOrganizationId: string): string {
    return `/api/v1/customer-organizations/${customerOrganizationId}/links`;
  }

  public list(customerOrganizationId: string): Observable<SidebarLink[]> {
    return this.httpClient.get<SidebarLink[]>(this.baseUrl(customerOrganizationId));
  }

  public create(customerOrganizationId: string, name: string, link: string): Observable<SidebarLink> {
    return this.httpClient.post<SidebarLink>(this.baseUrl(customerOrganizationId), {name, link});
  }

  public update(customerOrganizationId: string, linkId: string, name: string, link: string): Observable<SidebarLink> {
    return this.httpClient.put<SidebarLink>(`${this.baseUrl(customerOrganizationId)}/${linkId}`, {
      name,
      link,
    });
  }

  public delete(customerOrganizationId: string, linkId: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.baseUrl(customerOrganizationId)}/${linkId}`);
  }
}
