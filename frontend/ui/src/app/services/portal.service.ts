import {HttpBackend, HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {catchError, map, Observable, of, shareReplay} from 'rxjs';
import {Portal} from '../types/portal';

const defaultPortal: Portal = {
  customDomain: false,
  loginConfig: {
    registrationEnabled: false,
    oidcGithubEnabled: false,
    oidcGoogleEnabled: false,
    oidcMicrosoftEnabled: false,
    oidcGenericEnabled: false,
  },
};

/**
 * Owns the host-resolved bootstrap configuration of the unauthenticated pages: the portal branding of a custom app
 * domain and the login methods available on this host. The response depends only on the request host, so it is
 * requested once and replayed to every consumer.
 */
@Injectable({providedIn: 'root'})
export class PortalService {
  // Bypass global interceptors (auth, error toasts, maintenance-mode probe) so this best-effort
  // call stays silent and can never surface a toast or flip the app into maintenance mode.
  private readonly httpClient = new HttpClient(inject(HttpBackend));

  readonly portal$: Observable<Portal> = this.httpClient.get<Portal | null>('/api/public/v1/portal').pipe(
    // The endpoint answered custom domains with 204 before the login config moved into it, and its responses are
    // cached for a minute, so an empty body can still arrive from a cache while a new version rolls out.
    map((portal) => portal ?? defaultPortal),
    // best-effort: keep the Distr branding and offer email/password login only
    catchError(() => of(defaultPortal)),
    shareReplay(1)
  );

  readonly loginConfig = toSignal(this.portal$.pipe(map((portal) => portal.loginConfig)), {
    initialValue: defaultPortal.loginConfig,
  });
}
