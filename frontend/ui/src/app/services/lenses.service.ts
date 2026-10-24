import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {CustomerLens} from '../types/lens';

@Injectable({providedIn: 'root'})
export class LensesService {
  private readonly baseUrl = '/api/v1/lenses';
  private readonly httpClient = inject(HttpClient);

  public list(): Observable<CustomerLens[]> {
    return this.httpClient.get<CustomerLens[]>(this.baseUrl);
  }
}
