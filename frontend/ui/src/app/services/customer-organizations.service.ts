import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateUpdateCustomerOrganizationRequest,
  CustomerOrganization,
  CustomerOrganizationWithUsage,
} from '@distr-sh/distr-sdk';
import {Observable, shareReplay} from 'rxjs';

const baseUrl = '/api/v1/customer-organizations';

@Injectable({
  providedIn: 'root',
})
export class CustomerOrganizationsService {
  private readonly httpClient = inject(HttpClient);

  public getCustomerOrganizations(): Observable<CustomerOrganizationWithUsage[]> {
    return this.httpClient.get<CustomerOrganizationWithUsage[]>(baseUrl);
  }

  public getCustomerOrganizationById(id: string): Observable<CustomerOrganization> {
    return this.httpClient.get<CustomerOrganization>(`${baseUrl}/${id}`);
  }

  public createCustomerOrganization(
    request: CreateUpdateCustomerOrganizationRequest
  ): Observable<CustomerOrganization> {
    return this.httpClient.post<CustomerOrganization>(baseUrl, request);
  }

  public updateCustomerOrganization(
    id: string,
    request: CreateUpdateCustomerOrganizationRequest
  ): Observable<CustomerOrganization> {
    return this.httpClient.put<CustomerOrganization>(`${baseUrl}/${id}`, request);
  }

  public deleteCustomerOrganization(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }
}

@Injectable()
export class CustomerOrganizationsCache {
  private readonly svc = inject(CustomerOrganizationsService);

  private readonly upstream = this.svc.getCustomerOrganizations().pipe(shareReplay(1));

  public getCustomerOrganizations(): Observable<CustomerOrganizationWithUsage[]> {
    return this.upstream;
  }
}
