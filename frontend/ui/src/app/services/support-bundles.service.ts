import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateSupportBundleCommentRequest,
  CreateSupportBundleRequest,
  CreateSupportBundleResponse,
  CreateUpdateSupportBundleConfigurationRequest,
  SupportBundle,
  SupportBundleComment,
  SupportBundleConfigurationEnvVar,
  SupportBundleDetail,
  UpdateSupportBundleStatusRequest,
} from '../types/support-bundle';

const baseUrl = '/api/v1/support-bundles';

export function supportBundleZipFileName(bundle: SupportBundle): string {
  const part = (s: string) =>
    s
      .toLowerCase()
      .replaceAll(/[^a-z]/g, '')
      .substring(0, 16);
  const parts = ['distr-support-bundle'];
  const customer = part(bundle.customerOrganizationName || '');
  if (customer) {
    parts.push(customer);
  }
  const title = part(bundle.title || '');
  if (title) {
    parts.push(title);
  }
  parts.push(bundle.id.substring(0, 8));
  return parts.join('-') + '.zip';
}

@Injectable({providedIn: 'root'})
export class SupportBundlesService {
  private readonly httpClient = inject(HttpClient);

  public getConfiguration() {
    return this.httpClient.get<SupportBundleConfigurationEnvVar[]>(`${baseUrl}/configuration`);
  }

  public updateConfiguration(request: CreateUpdateSupportBundleConfigurationRequest) {
    return this.httpClient.put<SupportBundleConfigurationEnvVar[]>(`${baseUrl}/configuration`, request);
  }

  public list() {
    return this.httpClient.get<SupportBundle[]>(baseUrl);
  }

  public get(id: string) {
    return this.httpClient.get<SupportBundleDetail>(`${baseUrl}/${id}`);
  }

  public downloadResources(id: string) {
    return this.httpClient.get(`${baseUrl}/${id}/download`, {responseType: 'blob'});
  }

  public create(request: CreateSupportBundleRequest) {
    return this.httpClient.post<CreateSupportBundleResponse>(baseUrl, request);
  }

  public updateStatus(id: string, request: UpdateSupportBundleStatusRequest) {
    return this.httpClient.patch<void>(`${baseUrl}/${id}/status`, request);
  }

  public createComment(bundleId: string, request: CreateSupportBundleCommentRequest) {
    return this.httpClient.post<SupportBundleComment>(`${baseUrl}/${bundleId}/comments`, request);
  }
}
