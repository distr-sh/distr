import {HttpClient, HttpParams} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {
  ArtifactPullFilterOption,
  ArtifactVersionPull,
  ArtifactVersionPullFilterOptions,
} from '../types/artifact-version-pull';

export interface ArtifactPullFilters {
  before?: Date;
  after?: Date;
  count?: number;
  customerOrganizationId?: string;
  userAccountId?: string;
  remoteAddress?: string;
  artifactId?: string;
  artifactVersionId?: string;
}

@Injectable({providedIn: 'root'})
export class ArtifactPullsService {
  private readonly baseUrl = '/api/v1/artifact-pulls';
  private readonly httpClient = inject(HttpClient);

  public get(filters: ArtifactPullFilters = {}): Observable<ArtifactVersionPull[]> {
    return this.httpClient.get<ArtifactVersionPull[]>(this.baseUrl, {params: this.buildParams(filters)});
  }

  public getFilterOptions(): Observable<ArtifactVersionPullFilterOptions> {
    return this.httpClient.get<ArtifactVersionPullFilterOptions>(`${this.baseUrl}/filter-options`);
  }

  public getVersionOptions(artifactId: string): Observable<ArtifactPullFilterOption[]> {
    const params = new HttpParams().set('artifactId', artifactId);
    return this.httpClient.get<ArtifactPullFilterOption[]>(`${this.baseUrl}/filter-options/versions`, {params});
  }

  public export(filters: ArtifactPullFilters = {}): Observable<Blob> {
    return this.httpClient.get(`${this.baseUrl}/export`, {params: this.buildParams(filters), responseType: 'blob'});
  }

  private buildParams(filters: ArtifactPullFilters): HttpParams {
    let params = new HttpParams();
    if (filters.before) {
      params = params.set('before', filters.before.toISOString());
    }
    if (filters.after) {
      params = params.set('after', filters.after.toISOString());
    }
    if (filters.count) {
      params = params.set('count', filters.count);
    }
    if (filters.customerOrganizationId) {
      params = params.set('customerOrganizationId', filters.customerOrganizationId);
    }
    if (filters.userAccountId) {
      params = params.set('userAccountId', filters.userAccountId);
    }
    if (filters.remoteAddress) {
      params = params.set('remoteAddress', filters.remoteAddress);
    }
    if (filters.artifactId) {
      params = params.set('artifactId', filters.artifactId);
    }
    if (filters.artifactVersionId) {
      params = params.set('artifactVersionId', filters.artifactVersionId);
    }
    return params;
  }
}
