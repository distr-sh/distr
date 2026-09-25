import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {firstValueFrom, Observable, tap} from 'rxjs';
import {LicenseTemplate} from '../types/license-template';
import {DefaultReactiveList} from './cache';

@Injectable({providedIn: 'root'})
export class LicenseTemplatesService {
  private readonly http = inject(HttpClient);

  private readonly templatesUrl = '/api/v1/license-templates';
  private readonly cache = new DefaultReactiveList(this.http.get<LicenseTemplate[]>(this.templatesUrl));

  list(): Observable<LicenseTemplate[]> {
    return this.cache.get();
  }

  refresh(): Promise<LicenseTemplate[]> {
    return firstValueFrom(
      this.http.get<LicenseTemplate[]>(this.templatesUrl).pipe(tap((templates) => this.cache.reset(templates)))
    );
  }

  create(
    request: Pick<LicenseTemplate, 'name' | 'payloadTemplate' | 'expirationGracePeriodDays'>
  ): Observable<LicenseTemplate> {
    return this.http.post<LicenseTemplate>(this.templatesUrl, request).pipe(tap((t) => this.cache.save(t)));
  }

  update(request: LicenseTemplate): Observable<LicenseTemplate> {
    return this.http
      .put<LicenseTemplate>(`${this.templatesUrl}/${request.id}`, request)
      .pipe(tap((t) => this.cache.save(t)));
  }

  delete(template: LicenseTemplate): Observable<void> {
    return this.http.delete<void>(`${this.templatesUrl}/${template.id}`).pipe(tap(() => this.cache.remove(template)));
  }
}
