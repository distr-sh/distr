import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {CustomerSettings, UpdateCustomerSettingsRequest} from '../types/customer-settings';

const baseUrl = '/api/v1/customer-settings';

@Injectable({
  providedIn: 'root',
})
export class CustomerSettingsService {
  private readonly httpClient = inject(HttpClient);

  public get(customerOrganizationId: string): Observable<CustomerSettings> {
    return this.httpClient.get<CustomerSettings>(baseUrl, {params: {customerOrganizationId}});
  }

  public update(customerOrganizationId: string, request: UpdateCustomerSettingsRequest): Observable<CustomerSettings> {
    return this.httpClient.put<CustomerSettings>(baseUrl, request, {params: {customerOrganizationId}});
  }
}
