import {HttpClient, HttpErrorResponse, HttpParams} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  CreateDeploymentTargetRequest,
  DeploymentRequest,
  DeploymentRevisionResponse,
  DeploymentTarget,
  DeploymentTargetAccessResponse,
  PatchDeploymentRequest,
  UpdateDeploymentTargetRequest,
} from '@distr-sh/distr-sdk';
import {EMPTY, merge, Observable, retry, shareReplay, Subject, switchMap, tap, timer} from 'rxjs';
import {ReactiveList} from './cache';

export interface DeploymentTargetFilter {
  customerOrganizationIds: string[];
  internal: boolean;
}

class DeploymentTargetsReactiveList extends ReactiveList<DeploymentTarget> {
  protected override identify = (dt: DeploymentTarget) => dt.id;
  protected override sortAttr = (dt: DeploymentTarget) => dt.customerOrganization?.name ?? dt.name;
}

@Injectable({
  providedIn: 'root',
})
export class DeploymentTargetsService {
  private readonly deploymentTargetsBaseUrl = '/api/v1/deployment-targets';
  private readonly deploymentsBaseUrl = '/api/v1/deployments';
  private readonly httpClient = inject(HttpClient);
  private readonly cache = new DeploymentTargetsReactiveList(
    this.httpClient.get<DeploymentTarget[]>(this.deploymentTargetsBaseUrl)
  );

  private readonly pollRefresh$ = new Subject<void>();
  private readonly sharedPolling$ = this.createPolling(new HttpParams()).pipe(
    tap((dts) => this.cache.reset(dts)),
    shareReplay({
      bufferSize: 1,
      refCount: true,
    })
  );

  list(): Observable<DeploymentTarget[]> {
    return this.cache.get();
  }

  /**
   * Polls the deployment targets of the given customers plus, with `internal`, those of no customer, or all
   * of them when the filter is empty. Only the unfiltered poll updates the cached list.
   */
  poll(
    filter: DeploymentTargetFilter = {customerOrganizationIds: [], internal: false}
  ): Observable<DeploymentTarget[]> {
    if (filter.customerOrganizationIds.length === 0 && !filter.internal) {
      return this.sharedPolling$;
    }
    let params = new HttpParams();
    for (const id of filter.customerOrganizationIds) {
      params = params.append('customerOrganizationId', id);
    }
    if (filter.internal) {
      params = params.set('internal', true);
    }
    return this.createPolling(params);
  }

  private createPolling(params: HttpParams): Observable<DeploymentTarget[]> {
    return merge(timer(0, 5000), this.pollRefresh$).pipe(
      switchMap(() => this.httpClient.get<DeploymentTarget[]>(this.deploymentTargetsBaseUrl, {params})),
      retry({
        delay: (e, c) =>
          e instanceof HttpErrorResponse && (!e.status || e.status >= 500)
            ? timer(Math.min(Math.pow(c, 2), 30) * 1000)
            : EMPTY,
      })
    );
  }

  create(request: CreateDeploymentTargetRequest): Observable<DeploymentTarget> {
    return this.httpClient.post<DeploymentTarget>(this.deploymentTargetsBaseUrl, request).pipe(
      tap((it) => {
        this.cache.save(it);
        this.pollRefresh$.next();
      })
    );
  }

  update(id: string, request: UpdateDeploymentTargetRequest): Observable<DeploymentTarget> {
    return this.httpClient.put<DeploymentTarget>(`${this.deploymentTargetsBaseUrl}/${id}`, request).pipe(
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
    return this.httpClient
      .post<DeploymentTargetAccessResponse>(`${this.deploymentTargetsBaseUrl}/${deploymentTargetId}/access-request`, {})
      .pipe(tap(() => this.pollRefresh$.next()));
  }

  cancelReconnect(deploymentTargetId: string): Observable<void> {
    return this.httpClient
      .delete<void>(`${this.deploymentTargetsBaseUrl}/${deploymentTargetId}/access-request`)
      .pipe(tap(() => this.pollRefresh$.next()));
  }

  deploy(request: DeploymentRequest): Observable<void> {
    return this.httpClient.put<void>(this.deploymentsBaseUrl, request).pipe(tap(() => this.pollRefresh$.next()));
  }

  undeploy(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.deploymentsBaseUrl}/${id}`).pipe(tap(() => this.pollRefresh$.next()));
  }

  patchDeployment(id: string, patch: PatchDeploymentRequest): Observable<void> {
    return this.httpClient
      .patch<void>(`${this.deploymentsBaseUrl}/${id}`, patch)
      .pipe(tap(() => this.pollRefresh$.next()));
  }

  public getRevisions(deploymentId: string): Observable<DeploymentRevisionResponse[]> {
    return this.httpClient.get<DeploymentRevisionResponse[]>(`${this.deploymentsBaseUrl}/${deploymentId}/revisions`);
  }

  public getNotes(deploymentTargetId: string) {
    return this.httpClient.get<{notes: string}>(`${this.deploymentTargetsBaseUrl}/${deploymentTargetId}/notes`);
  }

  public saveNotes(deploymentTargetId: string, notes: string) {
    return this.httpClient.put<void>(`${this.deploymentTargetsBaseUrl}/${deploymentTargetId}/notes`, {notes});
  }
}
