import {HttpClient} from '@angular/common/http';
import {Injectable, inject} from '@angular/core';
import {Observable} from 'rxjs';
import {License} from '../types/license';

@Injectable({providedIn: 'root'})
export class LicensesService {
  private http = inject(HttpClient);

  list(): Observable<License[]> {
    return this.http.get<License[]>('/api/v1/licenses');
  }
}
