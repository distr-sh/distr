import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  AssignCustomerToPartnerRequest,
  CreateUpdateCustomerOrganizationRequest,
  CustomerOrganization,
  CustomerOrganizationWithUsage,
} from '@distr-sh/distr-sdk';
import {Observable, tap} from 'rxjs';
import {DefaultReactiveList} from './cache';
import {ContextService} from './context.service';

const baseUrl = '/api/v1/customer-organizations';

@Injectable({
  providedIn: 'root',
})
export class CustomerOrganizationsService {
  private readonly httpClient = inject(HttpClient);
  private readonly contextService = inject(ContextService);
  private readonly cache = new DefaultReactiveList(() => this.httpClient.get<CustomerOrganizationWithUsage[]>(baseUrl));

  public getCustomerOrganizations(): Observable<CustomerOrganizationWithUsage[]> {
    return this.cache.get();
  }

  public refresh(): Promise<CustomerOrganizationWithUsage[]> {
    return this.cache.refresh();
  }

  public createCustomerOrganization(
    request: CreateUpdateCustomerOrganizationRequest
  ): Observable<CustomerOrganization> {
    return this.httpClient.post<CustomerOrganization>(baseUrl, request).pipe(
      tap((it) => {
        this.cache.save({...it, userCount: 0, deploymentTargetCount: 0});
        this.contextService.reload();
      })
    );
  }

  public updateCustomerOrganization(
    id: string,
    request: CreateUpdateCustomerOrganizationRequest
  ): Observable<CustomerOrganization> {
    return this.httpClient
      .put<CustomerOrganization>(`${baseUrl}/${id}`, request)
      .pipe(tap((it) => this.saveUpdated(it)));
  }

  public assignCustomerToPartner(
    id: string,
    request: AssignCustomerToPartnerRequest
  ): Observable<CustomerOrganization> {
    return this.httpClient
      .put<CustomerOrganization>(`${baseUrl}/${id}/partner`, request)
      .pipe(tap((it) => this.saveUpdated(it)));
  }

  public deleteCustomerOrganization(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`).pipe(
      tap(() => {
        this.cache.remove({id} as CustomerOrganizationWithUsage);
        this.contextService.reload();
      })
    );
  }

  // The response of a write carries no usage counts, so the cached entry keeps its own.
  private saveUpdated(updated: CustomerOrganization) {
    this.cache.save({userCount: 0, deploymentTargetCount: 0, ...this.cache.find(updated.id), ...updated});
  }
}
