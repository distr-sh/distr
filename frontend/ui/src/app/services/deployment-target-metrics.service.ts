import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {map, Observable, shareReplay, switchMap, timer} from 'rxjs';
import {DeploymentResourceMetrics, DeploymentTargetLatestMetrics} from '../types/deployment-target-metrics';
import {trackPolling} from './error-toast.interceptor';

@Injectable({
  providedIn: 'root',
})
export class DeploymentTargetsMetricsService {
  private readonly deploymentTargetMetricsBaseUrl = '/api/v1/deployment-target-metrics';
  private readonly httpClient = inject(HttpClient);

  private readonly sharedPolling$ = trackPolling((context) =>
    timer(0, 30_000).pipe(
      switchMap(() =>
        this.httpClient.get<DeploymentTargetLatestMetrics[]>(this.deploymentTargetMetricsBaseUrl, {context})
      )
    )
  ).pipe(
    shareReplay({
      bufferSize: 1,
      refCount: true,
    })
  );

  poll(): Observable<DeploymentTargetLatestMetrics[]> {
    return this.sharedPolling$;
  }

  // The endpoint responds with 204 (and thus an empty body) when no metrics have been reported yet.
  pollDeploymentMetrics(deploymentId: string): Observable<DeploymentResourceMetrics | undefined> {
    return trackPolling((context) =>
      timer(0, 30_000).pipe(
        switchMap(() =>
          this.httpClient.get<DeploymentResourceMetrics | null>(`/api/v1/deployments/${deploymentId}/metrics`, {
            context,
          })
        ),
        map((metrics) => metrics ?? undefined)
      )
    );
  }
}
