import {HttpClient} from '@angular/common/http';
import {Injectable, inject} from '@angular/core';
import {Observable, Subject, switchMap, tap} from 'rxjs';
import {LicenseTemplate} from '../types/license-template';
import {DefaultReactiveList, ReactiveList} from './cache';

@Injectable({providedIn: 'root'})
export class LicenseTemplatesService {
  private readonly http = inject(HttpClient);

  private readonly cache: ReactiveList<LicenseTemplate>;
  private readonly templatesUrl = '/api/v1/license-templates';
  private readonly refresh$ = new Subject<void>();

  constructor() {
    this.cache = new DefaultReactiveList(this.http.get<LicenseTemplate[]>(this.templatesUrl));
    this.refresh$
      .pipe(
        switchMap(() => this.http.get<LicenseTemplate[]>(this.templatesUrl)),
        tap((templates) => this.cache.reset(templates))
      )
      .subscribe();
  }

  list(): Observable<LicenseTemplate[]> {
    return this.cache.get();
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
