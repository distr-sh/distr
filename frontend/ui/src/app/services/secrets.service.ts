import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {AffectedDeployment} from '../types/affected-deployment';
import {Secret} from '../types/secret';

const baseUrl = '/api/v1/secrets';

export interface UpdateSecretResponse extends Secret {
  affectedDeployments: AffectedDeployment[];
}

@Injectable({providedIn: 'root'})
export class SecretsService {
  private readonly httpClient = inject(HttpClient);

  public list(): Observable<Secret[]> {
    return this.httpClient.get<Secret[]>(baseUrl);
  }

  public create(key: string, value: string, customerOrganizationId?: string): Observable<Secret> {
    return this.httpClient.post<Secret>(baseUrl, {key, value, customerOrganizationId});
  }

  public update(id: string, value: string, confirm = false): Observable<UpdateSecretResponse> {
    return this.httpClient.put<UpdateSecretResponse>(
      `${baseUrl}/${id}`,
      {value},
      {
        params: confirm ? {confirm: 'true'} : {},
      }
    );
  }

  public delete(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }
}
