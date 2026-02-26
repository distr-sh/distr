import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateSupportBundleCommentRequest,
  CreateSupportBundleRequest,
  CreateSupportBundleResponse,
  CreateUpdateSupportBundleConfigurationRequest,
  SupportBundle,
  SupportBundleComment,
  SupportBundleConfiguration,
  SupportBundleDetail,
  SupportBundleResource,
  UpdateSupportBundleStatusRequest,
} from '../types/support-bundle';

const baseUrl = '/api/v1/support-bundles';

@Injectable({providedIn: 'root'})
export class SupportBundlesService {
  private readonly httpClient = inject(HttpClient);

  public getConfiguration() {
    return this.httpClient.get<SupportBundleConfiguration>(`${baseUrl}/configuration`);
  }

  public updateConfiguration(request: CreateUpdateSupportBundleConfigurationRequest) {
    return this.httpClient.put<SupportBundleConfiguration>(`${baseUrl}/configuration`, request);
  }

  public deleteConfiguration() {
    return this.httpClient.delete<void>(`${baseUrl}/configuration`);
  }

  public list() {
    return this.httpClient.get<SupportBundle[]>(baseUrl);
  }

  public get(id: string) {
    return this.httpClient.get<SupportBundleDetail>(`${baseUrl}/${id}`);
  }

  public create(request: CreateSupportBundleRequest) {
    return this.httpClient.post<CreateSupportBundleResponse>(baseUrl, request);
  }

  public updateStatus(id: string, request: UpdateSupportBundleStatusRequest) {
    return this.httpClient.patch<SupportBundle>(`${baseUrl}/${id}/status`, request);
  }

  public getResources(bundleId: string) {
    return this.httpClient.get<SupportBundleResource[]>(`${baseUrl}/${bundleId}/resources`);
  }

  public getComments(bundleId: string) {
    return this.httpClient.get<SupportBundleComment[]>(`${baseUrl}/${bundleId}/comments`);
  }

  public createComment(bundleId: string, request: CreateSupportBundleCommentRequest) {
    return this.httpClient.post<SupportBundleComment>(`${baseUrl}/${bundleId}/comments`, request);
  }
}
