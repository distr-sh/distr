import {HttpClient, HttpParams} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateUpdateVulnerabilityRequest,
  CreateVulnerabilityCommentRequest,
  UpdateVulnerabilityStatusRequest,
  Vulnerability,
  VulnerabilityDetail,
  VulnerabilityEvent,
  VulnerabilityFilter,
  VulnerabilityImpact,
} from '../types/vulnerability';

const baseUrl = '/api/v1/vulnerabilities';

@Injectable({providedIn: 'root'})
export class VulnerabilitiesService {
  private readonly httpClient = inject(HttpClient);

  public list(filter: VulnerabilityFilter = {}) {
    let params = new HttpParams();
    if (filter.status) {
      params = params.set('status', filter.status);
    }
    if (filter.severity) {
      params = params.set('severity', filter.severity);
    }
    if (filter.tag) {
      params = params.set('tag', filter.tag);
    }
    return this.httpClient.get<Vulnerability[]>(baseUrl, {params});
  }

  public get(id: string) {
    return this.httpClient.get<VulnerabilityDetail>(`${baseUrl}/${id}`);
  }

  public listTags() {
    return this.httpClient.get<string[]>(`${baseUrl}/tags`);
  }

  public getImpact(id: string) {
    return this.httpClient.get<VulnerabilityImpact>(`${baseUrl}/${id}/impact`);
  }

  public create(request: CreateUpdateVulnerabilityRequest) {
    return this.httpClient.post<VulnerabilityDetail>(baseUrl, request);
  }

  public update(id: string, request: CreateUpdateVulnerabilityRequest) {
    return this.httpClient.put<VulnerabilityDetail>(`${baseUrl}/${id}`, request);
  }

  public updateStatus(id: string, request: UpdateVulnerabilityStatusRequest) {
    return this.httpClient.patch<VulnerabilityDetail>(`${baseUrl}/${id}/status`, request);
  }

  public delete(id: string) {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }

  public createComment(id: string, request: CreateVulnerabilityCommentRequest) {
    return this.httpClient.post<VulnerabilityEvent>(`${baseUrl}/${id}/comments`, request);
  }
}
