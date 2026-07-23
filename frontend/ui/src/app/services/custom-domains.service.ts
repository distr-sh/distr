import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {CreateCustomDomainRequest, CustomDomain} from '../types/custom-domain';

const baseUrl = '/api/v1/custom-domains';

@Injectable({
  providedIn: 'root',
})
export class CustomDomainsService {
  private readonly httpClient = inject(HttpClient);

  public list(): Observable<CustomDomain[]> {
    return this.httpClient.get<CustomDomain[]>(baseUrl);
  }

  public create(request: CreateCustomDomainRequest): Observable<CustomDomain> {
    return this.httpClient.post<CustomDomain>(baseUrl, request);
  }

  public delete(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${baseUrl}/${id}`);
  }
}
