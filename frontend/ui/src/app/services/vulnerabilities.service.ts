import {HttpClient, HttpParams} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateUpdateVulnerabilityRequest,
  CreateVulnerabilityCommentRequest,
  CreateVulnerabilityRequest,
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
    for (const status of filter.status ?? []) {
      params = params.append('status', status);
    }
    for (const severity of filter.severity ?? []) {
      params = params.append('severity', severity);
    }
    for (const tag of filter.tag ?? []) {
      params = params.append('tag', tag);
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

  public create(request: CreateVulnerabilityRequest) {
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
