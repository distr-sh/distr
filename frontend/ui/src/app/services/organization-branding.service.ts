import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {OrganizationBranding} from '@distr-sh/distr-sdk';
import {BehaviorSubject, catchError, Observable, of, shareReplay, tap, throwError} from 'rxjs';

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

  private inFlight?: Observable<OrganizationBranding>;

  get(): Observable<OrganizationBranding> {
    const cached = this.brandingSubject.value;
    if (cached) {
      return of(cached);
    }
    // Callers on the same page (the navbar and the page it renders) subscribe before the first response arrives,
    // so the request itself has to be shared and not just its result.
    if (!this.inFlight) {
      this.inFlight = this.httpClient.get<OrganizationBranding>(this.organizationBrandingUrl).pipe(
        tap((branding) => {
          this.inFlight = undefined;
          this.brandingSubject.next(branding);
        }),
        catchError((err) => {
          this.inFlight = undefined;
          return throwError(() => err);
        }),
        shareReplay({bufferSize: 1, refCount: false})
      );
    }
    return this.inFlight;
  }

  upsert(organizationBranding: OrganizationBranding): Observable<OrganizationBranding> {
    return this.httpClient
      .put<OrganizationBranding>(this.organizationBrandingUrl, organizationBranding)
      .pipe(tap((obj) => this.brandingSubject.next(obj)));
  }
}
