import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateUpdatePartnerOrganizationRequest,
  PartnerOrganization,
  PartnerOrganizationWithUsage,
} from '@distr-sh/distr-sdk';
import {Observable} from 'rxjs';

const baseUrl = '/api/v1/partner-organizations';

@Injectable({
  providedIn: 'root',
})
export class PartnerOrganizationsService {
  private readonly httpClient = inject(HttpClient);

  public getPartnerOrganizations(): Observable<PartnerOrganizationWithUsage[]> {
    return this.httpClient.get<PartnerOrganizationWithUsage[]>(baseUrl);
  }

  public createPartnerOrganization(request: CreateUpdatePartnerOrganizationRequest): Observable<PartnerOrganization> {
    return this.httpClient.post<PartnerOrganization>(baseUrl, request);
  }

  public updatePartnerOrganization(
    id: string,
    request: CreateUpdatePartnerOrganizationRequest
  ): Observable<PartnerOrganization> {
    return this.httpClient.put<PartnerOrganization>(`${baseUrl}/${id}`, request);
  }

  public deletePartnerOrganization(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }
}
