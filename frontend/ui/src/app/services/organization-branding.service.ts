import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {OrganizationBranding} from '@distr-sh/distr-sdk';
import {BehaviorSubject, catchError, map, Observable, of, tap} from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class OrganizationBrandingService {
  private readonly httpClient = inject(HttpClient);

  private readonly organizationBrandingUrl = '/api/v1/organization/branding';
  // Holds the current branding so consumers (e.g. the navbar) update reactively when it is (re)loaded or saved.
  private readonly brandingSubject = new BehaviorSubject<OrganizationBranding | undefined>(undefined);

  /** Emits the current organization branding and every subsequent change (load or save). */
  readonly branding$ = this.brandingSubject.asObservable();

  get(): Observable<OrganizationBranding> {
    const cached = this.brandingSubject.value;
    if (cached) {
      return of(cached);
    }
    return this.httpClient
      .get<OrganizationBranding>(this.organizationBrandingUrl)
      .pipe(tap((branding) => this.brandingSubject.next(branding)));
  }

  upsert(organizationBranding: OrganizationBranding): Observable<OrganizationBranding> {
    return this.httpClient
      .put<OrganizationBranding>(this.organizationBrandingUrl, organizationBranding)
      .pipe(tap((obj) => this.brandingSubject.next(obj)));
  }

  registryDomain(): Observable<string | undefined> {
    return this.get().pipe(
      map((branding) => branding.registryDomain),
      catchError(() => of(undefined))
    );
  }
}
