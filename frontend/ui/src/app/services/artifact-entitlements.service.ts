import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {firstValueFrom, Observable, tap} from 'rxjs';
import {ArtifactEntitlement} from '../types/artifact-entitlement';
import {DefaultReactiveList} from './cache';
import {CrudService} from './interfaces';

@Injectable({providedIn: 'root'})
export class ArtifactEntitlementsService implements CrudService<ArtifactEntitlement> {
  private readonly http = inject(HttpClient);

  private readonly artifactEntitlementsUrl = '/api/v1/artifact-entitlements';
  private readonly cache = new DefaultReactiveList(this.http.get<ArtifactEntitlement[]>(this.artifactEntitlementsUrl));

  public list(): Observable<ArtifactEntitlement[]> {
    return this.cache.get();
  }

  refresh(): Promise<ArtifactEntitlement[]> {
    return firstValueFrom(
      this.http
        .get<ArtifactEntitlement[]>(this.artifactEntitlementsUrl)
        .pipe(tap((entitlements) => this.cache.reset(entitlements)))
    );
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
