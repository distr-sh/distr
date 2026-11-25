import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable, tap} from 'rxjs';
import {AffectedDeployment} from '../types/affected-deployment';
import {Secret} from '../types/secret';
import {ReactiveList} from './cache';
import {skipTrim} from './trim.interceptor';

const baseUrl = '/api/v1/secrets';

export interface UpdateSecretResponse extends Secret {
  affectedDeployments: AffectedDeployment[];
}

class SecretsReactiveList extends ReactiveList<Secret> {
  protected override identify = (s: Secret) => s.id;
  protected override sortAttr = (s: Secret) => s.key;
}

@Injectable({providedIn: 'root'})
export class SecretsService {
  private readonly httpClient = inject(HttpClient);
  private readonly cache = new SecretsReactiveList(() => this.httpClient.get<Secret[]>(baseUrl));

  public list(): Observable<Secret[]> {
    return this.cache.get();
  }

  public refresh(): Promise<Secret[]> {
    return this.cache.refresh();
  }

  public create(key: string, value: string, customerOrganizationId?: string): Observable<Secret> {
    return this.httpClient
      .post<Secret>(
        baseUrl,
        {key, value, customerOrganizationId},
        {
          context: skipTrim('value'),
        }
      )
      .pipe(tap((it) => this.cache.save(it)));
  }

  public update(id: string, value: string, confirm = false): Observable<UpdateSecretResponse> {
    return this.httpClient
      .put<UpdateSecretResponse>(
        `${baseUrl}/${id}`,
        {value},
        {
          context: skipTrim('value'),
          params: confirm ? {confirm: 'true'} : {},
        }
      )
      .pipe(tap(({affectedDeployments, ...secret}) => this.cache.save(secret)));
  }

  public delete(secret: Secret): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${secret.id}`).pipe(tap(() => this.cache.remove(secret)));
  }
}
