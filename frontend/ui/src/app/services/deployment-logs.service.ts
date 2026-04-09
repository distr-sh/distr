import {HttpClient, HttpParams} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {DeploymentLogRecord, DeploymentLogRecordResources} from '../types/deployment-log-record';
import {TimeseriesOptions, timeseriesOptionsAsParams} from '../types/timeseries-options';

function appendResources(params: HttpParams, resources: string[]): HttpParams {
  for (const r of resources) {
    params = params.append('resource', r);
  }
  return params;
}

@Injectable({providedIn: 'root'})
export class DeploymentLogsService {
  private readonly httpClient = inject(HttpClient);

  public getResources(deploymentId: string): Observable<DeploymentLogRecordResources> {
    return this.httpClient.get<DeploymentLogRecordResources>(`/api/v1/deployments/${deploymentId}/logs/resources`);
  }

  public get(
    deploymentId: string,
    resources: string[],
    options?: TimeseriesOptions
  ): Observable<DeploymentLogRecord[]> {
    const params = appendResources(new HttpParams({fromObject: timeseriesOptionsAsParams(options)}), resources);
    return this.httpClient.get<DeploymentLogRecord[]>(`/api/v1/deployments/${deploymentId}/logs`, {params});
  }

  public export(deploymentId: string, resources: string[]): Observable<Blob> {
    const params = appendResources(new HttpParams(), resources);
    return this.httpClient.get(`/api/v1/deployments/${deploymentId}/logs/export`, {params, responseType: 'blob'});
  }
}
