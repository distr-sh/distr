import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {License} from '../types/license';

@Injectable({providedIn: 'root'})
export class LicensesService {
  constructor(private http: HttpClient) {}

  list(): Observable<License[]> {
    return this.http.get<License[]>('/api/v1/licenses');
  }
}
