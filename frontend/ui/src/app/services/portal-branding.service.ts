import {DOCUMENT} from '@angular/common';
import {HttpBackend, HttpClient} from '@angular/common/http';
import {inject, Injectable, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {Title} from '@angular/platform-browser';
import {catchError, map, Observable, of, Subject, switchMap} from 'rxjs';
import {AuthService} from './auth.service';
import {OrganizationBrandingService} from './organization-branding.service';

interface PortalResponse {
  pageTitle?: string;
  faviconUrl?: string;
  logoUrl?: string;
}

interface ResolvedBranding {
  pageTitle?: string;
  faviconUrl?: string;
  logoUrl?: string;
  // Whether the request host matches a custom app domain. Used to drop Distr-specific branding (logo, website
  // links) on custom domains, even when no branding assets are configured.
  customDomain: boolean;
}

/**
 * Resolves the portal branding (browser tab title and favicon) and applies it to the document. Runs on app boot,
 * so it also affects the login page where no user is authenticated. Branding is resolved from two sources:
 *
 * 1. The custom app domain (host), which applies to everyone visiting it regardless of authentication.
 * 2. The organization the user is currently logged in with, which takes precedence once authenticated.
 */
@Injectable({providedIn: 'root'})
export class PortalBrandingService {
  // Bypass global interceptors (auth, error toasts, maintenance-mode probe) so this best-effort
  // call stays silent and can never surface a toast or flip the app into maintenance mode.
  private readonly httpClient = new HttpClient(inject(HttpBackend));
  private readonly title = inject(Title);
  private readonly document = inject(DOCUMENT);
  private readonly auth = inject(AuthService);
  private readonly organizationBrandingService = inject(OrganizationBrandingService);

  // Captured before any branding is applied so cleared/absent fields can be reset to the app defaults instead
  // of leaking a previously applied source's values.
  private readonly defaultTitle = this.title.getTitle();
  private readonly defaultIconLinks = Array.from(
    this.document.head.querySelectorAll<HTMLLinkElement>("link[rel~='icon']")
  ).map((link) => link.cloneNode(true) as HTMLLinkElement);

  // switchMap cancels any in-flight resolution when a newer apply() is triggered, so a slow request from an
  // earlier invocation can never overwrite branding applied by a later one (e.g. host branding started at
  // bootstrap resolving after the authenticated organization's branding was applied on login).
  private readonly applyTrigger = new Subject<void>();

  // Host-resolved portal logo and custom-domain flag for the (unauthenticated) login and related pages, so they
  // can replace the Distr logo and drop links to the Distr website on custom domains.
  private readonly logoUrlSignal = signal<string | undefined>(undefined);
  readonly logoUrl = this.logoUrlSignal.asReadonly();
  private readonly customDomainSignal = signal(false);
  readonly customDomain = this.customDomainSignal.asReadonly();

  constructor() {
    this.applyTrigger
      .pipe(
        switchMap(() => this.resolveBranding()),
        takeUntilDestroyed()
      )
      .subscribe((branding) => this.applyBranding(branding));
  }

  apply(): void {
    this.applyTrigger.next();
  }

  private resolveBranding(): Observable<ResolvedBranding> {
    // Logo and custom-domain flag are always host-based, so resolve the host first regardless of auth state.
    return this.resolveHostBranding().pipe(
      switchMap((host) => {
        // Once logged in, the organization's title and favicon fully replace the host's (never merged).
        if (!this.auth.getClaims()) {
          return of(host);
        }
        return this.organizationBrandingService.get().pipe(
          map((branding) => ({
            pageTitle: branding.pageTitle,
            // Served via the public API since the browser loads the favicon without auth.
            faviconUrl: branding.faviconImageId ? `/api/public/v1/files/${branding.faviconImageId}` : undefined,
            logoUrl: host.logoUrl,
            customDomain: host.customDomain,
          })),
          // best-effort: e.g. 404 when the organization has no branding configured
          catchError(() => of(host))
        );
      })
    );
  }

  private resolveHostBranding(): Observable<ResolvedBranding> {
    return this.httpClient.get<PortalResponse | null>('/api/public/v1/portal').pipe(
      // A response is only returned for a custom app domain, so its presence indicates a custom domain.
      map((portal) => ({
        pageTitle: portal?.pageTitle,
        faviconUrl: portal?.faviconUrl,
        logoUrl: portal?.logoUrl,
        customDomain: portal != null,
      })),
      // best-effort: keep the default title, favicon and logo
      catchError(() => of<ResolvedBranding>({customDomain: false}))
    );
  }

  private applyBranding({pageTitle, faviconUrl, logoUrl, customDomain}: ResolvedBranding): void {
    this.title.setTitle(pageTitle || this.defaultTitle);
    if (faviconUrl) {
      this.setFavicon(faviconUrl);
    } else {
      this.restoreDefaultFavicon();
    }
    this.logoUrlSignal.set(logoUrl);
    this.customDomainSignal.set(customDomain);
  }

  private setFavicon(url: string): void {
    const head = this.document.head;
    head.querySelectorAll("link[rel~='icon']").forEach((link) => link.remove());
    const link = this.document.createElement('link');
    link.rel = 'icon';
    link.href = url;
    head.appendChild(link);
  }

  private restoreDefaultFavicon(): void {
    const head = this.document.head;
    head.querySelectorAll("link[rel~='icon']").forEach((link) => link.remove());
    this.defaultIconLinks.forEach((link) => head.appendChild(link.cloneNode(true)));
  }
}
