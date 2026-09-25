import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {firstValueFrom, Observable, tap} from 'rxjs';
import {ApplicationEntitlement} from '../types/application-entitlement';
import {DefaultReactiveList} from './cache';
import {CrudService} from './interfaces';

@Injectable({
  providedIn: 'root',
})
export class ApplicationEntitlementsService implements CrudService<ApplicationEntitlement> {
  private readonly httpClient = inject(HttpClient);

  private readonly entitlementsUrl = '/api/v1/application-entitlements';
  private readonly cache = new DefaultReactiveList(this.httpClient.get<ApplicationEntitlement[]>(this.entitlementsUrl));

  list(applicationId?: string): Observable<ApplicationEntitlement[]> {
    if (applicationId) {
      return this.httpClient.get<ApplicationEntitlement[]>(this.entitlementsUrl, {params: {applicationId}});
    } else {
      return this.cache.get();
    }
  }

  refresh(): Promise<ApplicationEntitlement[]> {
    return firstValueFrom(
      this.httpClient
        .get<ApplicationEntitlement[]>(this.entitlementsUrl)
        .pipe(tap((entitlements) => this.cache.reset(entitlements)))
    );
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
