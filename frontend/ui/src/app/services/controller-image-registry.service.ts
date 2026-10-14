import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {ControllerImageRegistry, UpdateControllerImageRegistryRequest} from '../types/controller-image-registry';
import {customerScopeParams} from './customer-scope';

const baseUrl = '/api/v1/controller-image-registry';

@Injectable({providedIn: 'root'})
export class ControllerImageRegistryService {
  private readonly httpClient = inject(HttpClient);

  public get(customerOrganizationId?: string) {
    return this.httpClient.get<ControllerImageRegistry>(baseUrl, {params: customerScopeParams(customerOrganizationId)});
  }

  public update(request: UpdateControllerImageRegistryRequest) {
    return this.httpClient.put<ControllerImageRegistry>(baseUrl, request);
  }
}
