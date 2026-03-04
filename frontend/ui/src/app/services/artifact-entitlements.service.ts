import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable, Subject, switchMap, tap} from 'rxjs';
import {ArtifactEntitlement} from '../types/artifact-entitlement';
import {DefaultReactiveList, ReactiveList} from './cache';
import {CrudService} from './interfaces';

@Injectable({providedIn: 'root'})
export class ArtifactEntitlementsService implements CrudService<ArtifactEntitlement> {
  private readonly cache: ReactiveList<ArtifactEntitlement>;
  private readonly artifactEntitlementsUrl = '/api/v1/artifact-entitlements';
  private readonly refresh$ = new Subject<void>();

  constructor(private readonly http: HttpClient) {
    this.cache = new DefaultReactiveList(this.http.get<ArtifactEntitlement[]>(this.artifactEntitlementsUrl));
    this.refresh$
      .pipe(
        switchMap(() => this.http.get<ArtifactEntitlement[]>(this.artifactEntitlementsUrl)),
        tap((entitlements) => this.cache.reset(entitlements))
      )
      .subscribe();
  }

  public list(): Observable<ArtifactEntitlement[]> {
    return this.cache.get();
  }

  refresh() {
    this.refresh$.next();
  }

  create(request: ArtifactEntitlement): Observable<ArtifactEntitlement> {
    return this.http
      .post<ArtifactEntitlement>(this.artifactEntitlementsUrl, request)
      .pipe(tap((l) => this.cache.save(l)));
  }

  delete(request: ArtifactEntitlement): Observable<void> {
    return this.http
      .delete<void>(`${this.artifactEntitlementsUrl}/${request.id}`)
      .pipe(tap(() => this.cache.remove(request)));
  }

  update(request: ArtifactEntitlement): Observable<ArtifactEntitlement> {
    return this.http
      .put<ArtifactEntitlement>(`${this.artifactEntitlementsUrl}/${request.id}`, request)
      .pipe(tap((l) => this.cache.save(l)));
  }
}
