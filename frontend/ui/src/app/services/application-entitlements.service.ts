import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable, Subject, switchMap, tap} from 'rxjs';
import {ApplicationEntitlement} from '../types/application-entitlement';
import {DefaultReactiveList, ReactiveList} from './cache';
import {CrudService} from './interfaces';

@Injectable({
  providedIn: 'root',
})
export class ApplicationEntitlementsService implements CrudService<ApplicationEntitlement> {
  private readonly entitlementsUrl = '/api/v1/application-entitlements';
  private readonly cache: ReactiveList<ApplicationEntitlement>;
  private readonly refresh$ = new Subject<void>();

  constructor(private readonly httpClient: HttpClient) {
    this.cache = new DefaultReactiveList(this.httpClient.get<ApplicationEntitlement[]>(this.entitlementsUrl));
    this.refresh$
      .pipe(
        switchMap(() => this.httpClient.get<ApplicationEntitlement[]>(this.entitlementsUrl)),
        tap((entitlements) => this.cache.reset(entitlements))
      )
      .subscribe();
  }

  list(applicationId?: string): Observable<ApplicationEntitlement[]> {
    if (applicationId) {
      return this.httpClient.get<ApplicationEntitlement[]>(this.entitlementsUrl, {params: {applicationId}});
    } else {
      return this.cache.get();
    }
  }

  refresh() {
    this.refresh$.next();
  }

  create(entitlement: ApplicationEntitlement): Observable<ApplicationEntitlement> {
    return this.httpClient
      .post<ApplicationEntitlement>(this.entitlementsUrl, entitlement)
      .pipe(tap((it) => this.cache.save(it)));
  }

  update(entitlement: ApplicationEntitlement): Observable<ApplicationEntitlement> {
    return this.httpClient
      .put<ApplicationEntitlement>(`${this.entitlementsUrl}/${entitlement.id}`, entitlement)
      .pipe(tap((it) => this.cache.save(it)));
  }

  delete(entitlement: ApplicationEntitlement): Observable<void> {
    return this.httpClient
      .delete<void>(`${this.entitlementsUrl}/${entitlement.id}`)
      .pipe(tap(() => this.cache.remove(entitlement)));
  }
}
