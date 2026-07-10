import {DOCUMENT} from '@angular/common';
import {HttpBackend, HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Title} from '@angular/platform-browser';
import {firstValueFrom} from 'rxjs';
import {AuthService} from './auth.service';
import {OrganizationBrandingService} from './organization-branding.service';

interface PortalResponse {
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

  async apply(): Promise<void> {
    // Host-based branding applies to everyone on a custom app domain, including the (unauthenticated) login page.
    await this.applyHostBranding();
    // The current organization's branding takes precedence once the user is logged in.
    await this.applyContextBranding();
  }

  private async applyHostBranding(): Promise<void> {
    try {
      const portal = await firstValueFrom(this.httpClient.get<PortalResponse | null>('/api/public/v1/portal'));
      this.applyBranding(portal?.pageTitle, portal?.faviconUrl);
    } catch (e) {
      // best-effort: keep the default title and favicon
    }
  }

  private async applyContextBranding(): Promise<void> {
    if (!this.auth.getClaims()) {
      return;
    }
    try {
      const branding = await firstValueFrom(this.organizationBrandingService.get());
      // The favicon is loaded by the browser as a plain resource (no auth), so it is served via the public API.
      const faviconUrl = branding.faviconImageId ? `/api/public/v1/files/${branding.faviconImageId}` : undefined;
      this.applyBranding(branding.pageTitle, faviconUrl);
    } catch (e) {
      // best-effort: e.g. 404 when the organization has no branding configured
    }
  }

  private applyBranding(pageTitle?: string, faviconUrl?: string): void {
    if (pageTitle) {
      this.title.setTitle(pageTitle);
    }
    if (faviconUrl) {
      this.setFavicon(faviconUrl);
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
}
