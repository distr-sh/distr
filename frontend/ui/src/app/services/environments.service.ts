import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {CreateUpdateEnvironmentRequest, Environment} from '../types/environment';

const baseUrl = '/api/v1/environments';

@Injectable({
  providedIn: 'root',
})
export class EnvironmentsService {
  private readonly httpClient = inject(HttpClient);

  list(): Observable<Environment[]> {
    return this.httpClient.get<Environment[]>(baseUrl);
  }

  get(id: string): Observable<Environment> {
    return this.httpClient.get<Environment>(`${baseUrl}/${id}`);
  }

  create(request: CreateUpdateEnvironmentRequest): Observable<Environment> {
    return this.httpClient.post<Environment>(baseUrl, request);
  }

  update(id: string, request: CreateUpdateEnvironmentRequest): Observable<Environment> {
    return this.httpClient.put<Environment>(`${baseUrl}/${id}`, request);
  }

  delete(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }
}
