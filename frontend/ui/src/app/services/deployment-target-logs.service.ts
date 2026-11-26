import {HttpClient, HttpContext} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {DeploymentTargetLogRecord} from '../types/deployment-target-log-record';
import {TimeseriesOptions, timeseriesOptionsAsParams} from '../types/timeseries-options';

@Injectable({providedIn: 'root'})
export class DeploymentTargetLogsService {
  private readonly httpClient = inject(HttpClient);

  public get(
    deploymentTargetId: string,
    options?: TimeseriesOptions,
    context?: HttpContext
  ): Observable<DeploymentTargetLogRecord[]> {
    const params = {...timeseriesOptionsAsParams(options)};
    return this.httpClient.get<DeploymentTargetLogRecord[]>(`/api/v1/deployment-targets/${deploymentTargetId}/logs`, {
      params,
      context,
    });
  }

  public export(deploymentTargetId: string, options?: TimeseriesOptions): Observable<Blob> {
    const params = {...timeseriesOptionsAsParams(options)};
    return this.httpClient.get(`/api/v1/deployment-targets/${deploymentTargetId}/logs/export`, {
      params,
      responseType: 'blob',
    });
  }
}
