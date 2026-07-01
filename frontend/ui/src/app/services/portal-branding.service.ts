import {DOCUMENT} from '@angular/common';
import {HttpBackend, HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Title} from '@angular/platform-browser';
import {firstValueFrom} from 'rxjs';

interface PortalResponse {
  pageTitle?: string;
  faviconUrl?: string;
}

/**
 * Resolves the host-based portal branding (browser tab title and favicon) for the current custom app domain and
 * applies it to the document. Runs on app boot so it also affects the login page, where no user is authenticated.
 */
@Injectable({providedIn: 'root'})
export class PortalBrandingService {
  // Bypass global interceptors (auth, error toasts, maintenance-mode probe) so this best-effort
  // call stays silent and can never surface a toast or flip the app into maintenance mode.
  private readonly httpClient = new HttpClient(inject(HttpBackend));
  private readonly title = inject(Title);
  private readonly document = inject(DOCUMENT);

  async apply(): Promise<void> {
    try {
      const portal = await firstValueFrom(this.httpClient.get<PortalResponse>('/api/public/v1/portal'));
      if (portal.pageTitle) {
        this.title.setTitle(portal.pageTitle);
      }
      if (portal.faviconUrl) {
        this.setFavicon(portal.faviconUrl);
      }
    } catch {
      // best-effort: keep the default title and favicon
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
