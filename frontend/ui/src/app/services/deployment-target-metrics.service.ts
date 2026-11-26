import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {map, Observable, shareReplay, switchMap, timer} from 'rxjs';
import {DeploymentResourceMetrics, DeploymentTargetLatestMetrics} from '../types/deployment-target-metrics';
import {polling} from './error-toast.interceptor';

@Injectable({
  providedIn: 'root',
})
export class DeploymentTargetsMetricsService {
  private readonly deploymentTargetMetricsBaseUrl = '/api/v1/deployment-target-metrics';
  private readonly httpClient = inject(HttpClient);

  private readonly sharedPolling$ = timer(0, 30_000).pipe(
    switchMap(() =>
      this.httpClient.get<DeploymentTargetLatestMetrics[]>(this.deploymentTargetMetricsBaseUrl, {context: polling()})
    ),
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
    return timer(0, 30_000).pipe(
      switchMap(() =>
        this.httpClient.get<DeploymentResourceMetrics | null>(`/api/v1/deployments/${deploymentId}/metrics`, {
          context: polling(),
        })
      ),
      map((metrics) => metrics ?? undefined)
    );
  }
}
