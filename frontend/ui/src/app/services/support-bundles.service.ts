import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateSupportBundleCommentRequest,
  CreateSupportBundleRequest,
  CreateSupportBundleResponse,
  CreateUpdateSupportBundleConfigurationRequest,
  CreateUpdateSupportBundleConfigurationScriptRequest,
  SupportBundle,
  SupportBundleComment,
  SupportBundleConfigurationEnvVar,
  SupportBundleConfigurationScript,
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

  public getScripts() {
    return this.httpClient.get<SupportBundleConfigurationScript[]>(`${baseUrl}/configuration/scripts`);
  }

  public createScript(request: CreateUpdateSupportBundleConfigurationScriptRequest) {
    return this.httpClient.post<SupportBundleConfigurationScript>(`${baseUrl}/configuration/scripts`, request);
  }

  public updateScript(id: string, request: CreateUpdateSupportBundleConfigurationScriptRequest) {
    return this.httpClient.put<SupportBundleConfigurationScript>(`${baseUrl}/configuration/scripts/${id}`, request);
  }

  public deleteScript(id: string) {
    return this.httpClient.delete<void>(`${baseUrl}/configuration/scripts/${id}`);
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
