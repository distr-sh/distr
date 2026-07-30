import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {DeploymentRevisionStatus} from '@distr-sh/distr-sdk';
import {Observable} from 'rxjs';
import {TimeseriesOptions, timeseriesOptionsAsParams} from '../types/timeseries-options';

@Injectable({
  providedIn: 'root',
})
export class DeploymentStatusService {
  private readonly baseUrl = '/api/v1/deployments';
  private readonly httpClient = inject(HttpClient);

  getStatuses(deploymentId: string, options?: TimeseriesOptions): Observable<DeploymentRevisionStatus[]> {
    const params = timeseriesOptionsAsParams(options);
    return this.httpClient.get<DeploymentRevisionStatus[]>(`${this.baseUrl}/${deploymentId}/status`, {params});
  }

  export(deploymentId: string, options?: TimeseriesOptions): Observable<Blob> {
    const params = timeseriesOptionsAsParams(options);
    return this.httpClient.get(`${this.baseUrl}/${deploymentId}/status/export`, {params, responseType: 'blob'});
  }
}
