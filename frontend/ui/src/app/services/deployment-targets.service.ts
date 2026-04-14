import {HttpClient, HttpErrorResponse} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {DeploymentRequest, DeploymentTarget, DeploymentTargetAccessResponse} from '@distr-sh/distr-sdk';
import {EMPTY, merge, Observable, retry, shareReplay, Subject, switchMap, tap, timer} from 'rxjs';
import {ReactiveList} from './cache';
import {CrudService} from './interfaces';

class DeploymentTargetsReactiveList extends ReactiveList<DeploymentTarget> {
  protected override identify = (dt: DeploymentTarget) => dt.id;
  protected override sortAttr = (dt: DeploymentTarget) => dt.customerOrganization?.name ?? dt.name;
}

@Injectable({
  providedIn: 'root',
})
export class DeploymentTargetsService implements CrudService<DeploymentTarget> {
  private readonly deploymentTargetsBaseUrl = '/api/v1/deployment-targets';
  private readonly deploymentsBaseUrl = '/api/v1/deployments';
  private readonly httpClient = inject(HttpClient);
  private readonly cache = new DeploymentTargetsReactiveList(
    this.httpClient.get<DeploymentTarget[]>(this.deploymentTargetsBaseUrl)
  );

  private readonly pollRefresh$ = new Subject<void>();
  private readonly sharedPolling$ = merge(timer(0, 5000), this.pollRefresh$).pipe(
    switchMap(() => this.httpClient.get<DeploymentTarget[]>(this.deploymentTargetsBaseUrl)),
    tap((dts) => this.cache.reset(dts)),
    retry({
      delay: (e, c) =>
        e instanceof HttpErrorResponse && (!e.status || e.status >= 500)
          ? timer(Math.min(Math.pow(c, 2), 30) * 1000)
          : EMPTY,
    }),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    })
  );

  list(): Observable<DeploymentTarget[]> {
    return this.cache.get();
  }

  poll(): Observable<DeploymentTarget[]> {
    return this.sharedPolling$;
  }

  create(request: DeploymentTarget): Observable<DeploymentTarget> {
    return this.httpClient.post<DeploymentTarget>(this.deploymentTargetsBaseUrl, request).pipe(
      tap((it) => {
        this.cache.save(it);
        this.pollRefresh$.next();
      })
    );
  }

  update(request: DeploymentTarget): Observable<DeploymentTarget> {
    return this.httpClient.put<DeploymentTarget>(`${this.deploymentTargetsBaseUrl}/${request.id}`, request).pipe(
      tap((it) => {
        this.cache.save(it);
        this.pollRefresh$.next();
      })
    );
  }

  delete(request: DeploymentTarget): Observable<void> {
    return this.httpClient.delete<void>(`${this.deploymentTargetsBaseUrl}/${request.id}`).pipe(
      tap(() => {
        this.cache.remove(request);
        this.pollRefresh$.next();
      })
    );
  }

  requestAccess(deploymentTargetId: string) {
    return this.httpClient.post<DeploymentTargetAccessResponse>(
      `${this.deploymentTargetsBaseUrl}/${deploymentTargetId}/access-request`,
      {}
    );
  }

  deploy(request: DeploymentRequest): Observable<void> {
    return this.httpClient.put<void>(this.deploymentsBaseUrl, request).pipe(tap(() => this.pollRefresh$.next()));
  }

  undeploy(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.deploymentsBaseUrl}/${id}`).pipe(tap(() => this.pollRefresh$.next()));
  }

  public getNotes(deploymentTargetId: string) {
    return this.httpClient.get<{notes: string}>(`${this.deploymentTargetsBaseUrl}/${deploymentTargetId}/notes`);
  }

  public saveNotes(deploymentTargetId: string, notes: string) {
    return this.httpClient.put<void>(`${this.deploymentTargetsBaseUrl}/${deploymentTargetId}/notes`, {notes});
  }
}
