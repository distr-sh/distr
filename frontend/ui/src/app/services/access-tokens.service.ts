import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {
  AccessToken,
  AccessTokenSecretSlot,
  AccessTokenWithKey,
  CreateAccessTokenRequest,
  CreateAccessTokenSecretRequest,
  PatchAccessTokenRequest,
} from '../types/access-token';

const baseUrl = '/api/v1/settings/tokens';

@Injectable({providedIn: 'root'})
export class AccessTokensService {
  private readonly httpClient = inject(HttpClient);

  public list(): Observable<AccessToken[]> {
    return this.httpClient.get<AccessToken[]>(baseUrl);
  }

  public create(request: CreateAccessTokenRequest): Observable<AccessTokenWithKey> {
    return this.httpClient.post<AccessTokenWithKey>(baseUrl, request);
  }

  public patch(id: string, request: PatchAccessTokenRequest): Observable<AccessToken> {
    return this.httpClient.patch<AccessToken>(`${baseUrl}/${id}`, request);
  }

  public delete(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }

  public createSecret(id: string, request: CreateAccessTokenSecretRequest): Observable<AccessTokenWithKey> {
    return this.httpClient.post<AccessTokenWithKey>(`${baseUrl}/${id}/secrets`, request);
  }

  public deleteSecret(id: string, slot: AccessTokenSecretSlot): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}/secrets/${slot}`);
  }
}
