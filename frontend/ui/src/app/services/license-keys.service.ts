import {HttpClient} from '@angular/common/http';
import {Injectable, inject} from '@angular/core';
import {Observable, Subject, switchMap, tap} from 'rxjs';
import {AffectedDeployment} from '../types/affected-deployment';
import {CreateLicenseKeyRequest, LicenseKey, LicenseKeyRevision, UpdateLicenseKeyRequest} from '../types/license-key';
import {DefaultReactiveList, ReactiveList} from './cache';

export interface UpdateLicenseKeyResponse extends LicenseKey {
  affectedDeployments: AffectedDeployment[];
}

@Injectable({providedIn: 'root'})
export class LicenseKeysService {
  private readonly http = inject(HttpClient);

  private readonly cache: ReactiveList<LicenseKey>;
  private readonly licenseKeysUrl = '/api/v1/license-keys';
  private readonly refresh$ = new Subject<void>();

  constructor() {
    this.cache = new DefaultReactiveList(this.http.get<LicenseKey[]>(this.licenseKeysUrl));
    this.refresh$
      .pipe(
        switchMap(() => this.http.get<LicenseKey[]>(this.licenseKeysUrl)),
        tap((keys) => this.cache.reset(keys))
      )
      .subscribe();
  }

  public list(): Observable<LicenseKey[]> {
    return this.cache.get();
  }

  refresh() {
    this.refresh$.next();
  }

  create(request: CreateLicenseKeyRequest): Observable<LicenseKey> {
    return this.http.post<LicenseKey>(this.licenseKeysUrl, request).pipe(tap((l) => this.cache.save(l)));
  }

  update(id: string, request: UpdateLicenseKeyRequest, confirm = false): Observable<UpdateLicenseKeyResponse> {
    return this.http
      .put<UpdateLicenseKeyResponse>(`${this.licenseKeysUrl}/${id}`, request, {
        params: confirm ? {confirm: 'true'} : {},
      })
      .pipe(tap((response) => this.cache.save(response)));
  }

  delete(request: LicenseKey): Observable<void> {
    return this.http.delete<void>(`${this.licenseKeysUrl}/${request.id}`).pipe(tap(() => this.cache.remove(request)));
  }

  getToken(id: string): Observable<{token: string}> {
    return this.http.get<{token: string}>(`${this.licenseKeysUrl}/${id}/token`);
  }

  getRevisions(id: string): Observable<LicenseKeyRevision[]> {
    return this.http.get<LicenseKeyRevision[]>(`${this.licenseKeysUrl}/${id}/revisions`);
  }
}
