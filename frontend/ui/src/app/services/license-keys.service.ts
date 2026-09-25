import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {firstValueFrom, Observable, tap} from 'rxjs';
import {AffectedDeployment} from '../types/affected-deployment';
import {CreateLicenseKeyRequest, LicenseKey, LicenseKeyRevision, UpdateLicenseKeyRequest} from '../types/license-key';
import {DefaultReactiveList} from './cache';

export interface UpdateLicenseKeyResponse extends LicenseKey {
  affectedDeployments: AffectedDeployment[];
}

@Injectable({providedIn: 'root'})
export class LicenseKeysService {
  private readonly http = inject(HttpClient);

  private readonly licenseKeysUrl = '/api/v1/license-keys';
  private readonly cache = new DefaultReactiveList(this.http.get<LicenseKey[]>(this.licenseKeysUrl));

  public list(): Observable<LicenseKey[]> {
    return this.cache.get();
  }

  refresh(): Promise<LicenseKey[]> {
    return firstValueFrom(this.http.get<LicenseKey[]>(this.licenseKeysUrl).pipe(tap((keys) => this.cache.reset(keys))));
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
