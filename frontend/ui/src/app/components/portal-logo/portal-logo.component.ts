import {ChangeDetectionStrategy, Component, computed, inject, input} from '@angular/core';
import {WEBSITE_URL} from '../../../constants';
import {PortalBrandingService} from '../../services/portal-branding.service';

/**
 * Renders the branding logo used in the header of the login and related (unauthenticated) pages. On a custom app
 * domain it shows the organization's portal logo instead of the Distr logo and never links to the Distr website.
 */
@Component({
  selector: 'app-portal-logo',
  templateUrl: './portal-logo.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PortalLogoComponent {
  private readonly portalBranding = inject(PortalBrandingService);

  /** Links the Distr wordmark to the Distr website (only on the default, non-custom-domain deployment). */
  public readonly linkToWebsite = input(false);

  protected readonly websiteUrl = WEBSITE_URL;
  protected readonly logoUrl = this.portalBranding.logoUrl;
  protected readonly customDomain = this.portalBranding.customDomain;

  protected readonly showCustomLogo = computed(() => this.customDomain() && !!this.logoUrl());
  protected readonly showWebsiteLink = computed(() => !this.customDomain() && this.linkToWebsite());
}
