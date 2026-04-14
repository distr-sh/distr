import {HttpClient} from '@angular/common/http';
import {Injectable, inject} from '@angular/core';
import {OrganizationBranding} from '@distr-sh/distr-sdk';
import {Observable, of, tap} from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class OrganizationBrandingService {
  private readonly httpClient = inject(HttpClient);

  private readonly organizationBrandingUrl = '/api/v1/organization/branding';
  private cache?: OrganizationBranding;

  get(): Observable<OrganizationBranding> {
    if (this.cache) {
      return of(this.cache);
    }
    return this.httpClient
      .get<OrganizationBranding>(this.organizationBrandingUrl)
      .pipe(tap((branding) => (this.cache = branding)));
  }

  create(organizationBranding: FormData): Observable<OrganizationBranding> {
    return this.httpClient
      .post<OrganizationBranding>(this.organizationBrandingUrl, organizationBranding)
      .pipe(tap((obj) => (this.cache = obj)));
  }

  update(organizationBranding: FormData): Observable<OrganizationBranding> {
    return this.httpClient
      .put<OrganizationBranding>(this.organizationBrandingUrl, organizationBranding)
      .pipe(tap((obj) => (this.cache = obj)));
  }
}
