import {DOCUMENT} from '@angular/common';
import {HttpBackend, HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {Title} from '@angular/platform-browser';
import {catchError, map, Observable, of, Subject, switchMap} from 'rxjs';
import {AuthService} from './auth.service';
import {OrganizationBrandingService} from './organization-branding.service';

interface PortalResponse {
  pageTitle?: string;
  faviconUrl?: string;
}

interface ResolvedBranding {
  pageTitle?: string;
  faviconUrl?: string;
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
    // The current organization's branding takes full precedence once the user is logged in, so it fully
    // replaces (never merges with) any host-based branding to avoid mixing two organizations' branding. When
    // it is unavailable (logged out, or no branding configured) we fall back to host-based branding, which
    // applies to everyone on a custom app domain, including the (unauthenticated) login page.
    if (!this.auth.getClaims()) {
      return this.resolveHostBranding();
    }
    return this.organizationBrandingService.get().pipe(
      map((branding) => ({
        pageTitle: branding.pageTitle,
        // The favicon is loaded by the browser as a plain resource (no auth), so it is served via the public API.
        faviconUrl: branding.faviconImageId ? `/api/public/v1/files/${branding.faviconImageId}` : undefined,
      })),
      // best-effort: e.g. 404 when the organization has no branding configured
      catchError(() => this.resolveHostBranding())
    );
  }

  private resolveHostBranding(): Observable<ResolvedBranding> {
    return this.httpClient.get<PortalResponse | null>('/api/public/v1/portal').pipe(
      map((portal) => ({pageTitle: portal?.pageTitle, faviconUrl: portal?.faviconUrl})),
      // best-effort: keep the default title and favicon
      catchError(() => of<ResolvedBranding>({}))
    );
  }

  private applyBranding({pageTitle, faviconUrl}: ResolvedBranding): void {
    this.title.setTitle(pageTitle || this.defaultTitle);
    if (faviconUrl) {
      this.setFavicon(faviconUrl);
    } else {
      this.restoreDefaultFavicon();
    }
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
