import {effect, inject, Injectable, signal} from '@angular/core';
import {Title} from '@angular/platform-browser';
import {RouterStateSnapshot, TitleStrategy} from '@angular/router';
import {PortalBrandingService} from './portal-branding.service';

@Injectable({providedIn: 'root'})
export class PageTitleStrategy extends TitleStrategy {
  private readonly title = inject(Title);
  private readonly portalBranding = inject(PortalBrandingService);
  private readonly routeTitle = signal<string | undefined>(undefined);

  constructor() {
    super();
    effect(() => {
      const brand = this.portalBranding.pageTitle();
      const route = this.routeTitle();
      this.title.setTitle(route ? `${route} | ${brand}` : brand);
    });
  }

  override updateTitle(snapshot: RouterStateSnapshot): void {
    this.routeTitle.set(this.buildTitle(snapshot));
  }
}
