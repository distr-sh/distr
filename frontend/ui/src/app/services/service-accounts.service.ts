import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  AccessToken,
  AccessTokenWithKey,
  CreateAccessTokenRequest,
  CreateServiceAccountRequest,
  PatchServiceAccountRequest,
  ServiceAccount,
} from '@distr-sh/distr-sdk';
import {Observable} from 'rxjs';

@Injectable({providedIn: 'root'})
export class ServiceAccountsService {
  private readonly baseUrl = '/api/v1/service-accounts';
  private readonly httpClient = inject(HttpClient);

  public list(): Observable<ServiceAccount[]> {
    return this.httpClient.get<ServiceAccount[]>(this.baseUrl);
  }

  public get(id: string): Observable<ServiceAccount> {
    return this.httpClient.get<ServiceAccount>(`${this.baseUrl}/${id}`);
  }

  public create(request: CreateServiceAccountRequest): Observable<ServiceAccount> {
    return this.httpClient.post<ServiceAccount>(this.baseUrl, request);
  }

  public patch(id: string, request: PatchServiceAccountRequest): Observable<ServiceAccount> {
    return this.httpClient.patch<ServiceAccount>(`${this.baseUrl}/${id}`, request);
  }

  public delete(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.baseUrl}/${id}`);
  }

  public listTokens(serviceAccountId: string): Observable<AccessToken[]> {
    return this.httpClient.get<AccessToken[]>(`${this.baseUrl}/${serviceAccountId}/tokens`);
  }

  public createToken(serviceAccountId: string, request: CreateAccessTokenRequest): Observable<AccessTokenWithKey> {
    return this.httpClient.post<AccessTokenWithKey>(`${this.baseUrl}/${serviceAccountId}/tokens`, request);
  }

  public deleteToken(serviceAccountId: string, tokenId: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.baseUrl}/${serviceAccountId}/tokens/${tokenId}`);
  }
}
