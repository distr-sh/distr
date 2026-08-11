import {HttpClient, HttpParams} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  Advisory,
  AdvisoryDetail,
  AdvisoryEvent,
  AdvisoryFilter,
  AdvisoryImpact,
  CreateAdvisoryCommentRequest,
  CreateAdvisoryRequest,
  CreateUpdateAdvisoryRequest,
  UpdateAdvisoryStatusRequest,
} from '../types/advisory';

const baseUrl = '/api/v1/advisories';

@Injectable({providedIn: 'root'})
export class AdvisoriesService {
  private readonly httpClient = inject(HttpClient);

  public list(filter: AdvisoryFilter = {}) {
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
    return this.httpClient.get<Advisory[]>(baseUrl, {params});
  }

  public get(id: string) {
    return this.httpClient.get<AdvisoryDetail>(`${baseUrl}/${id}`);
  }

  public listTags() {
    return this.httpClient.get<string[]>(`${baseUrl}/tags`);
  }

  public getImpact(id: string) {
    return this.httpClient.get<AdvisoryImpact>(`${baseUrl}/${id}/impact`);
  }

  public create(request: CreateAdvisoryRequest) {
    return this.httpClient.post<AdvisoryDetail>(baseUrl, request);
  }

  public update(id: string, request: CreateUpdateAdvisoryRequest) {
    return this.httpClient.put<AdvisoryDetail>(`${baseUrl}/${id}`, request);
  }

  public updateStatus(id: string, request: UpdateAdvisoryStatusRequest) {
    return this.httpClient.patch<AdvisoryDetail>(`${baseUrl}/${id}/status`, request);
  }

  public createComment(id: string, request: CreateAdvisoryCommentRequest) {
    return this.httpClient.post<AdvisoryEvent>(`${baseUrl}/${id}/comments`, request);
  }
}
