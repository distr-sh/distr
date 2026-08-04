export interface PortalOIDCProvider {
  id: string;
  name: string;
  // Whether the login page redirects to this provider without user interaction.
  spInitiated: boolean;
}

export interface PortalLoginConfig {
  registrationEnabled: boolean;
  oidcGithubEnabled: boolean;
  oidcGoogleEnabled: boolean;
  oidcMicrosoftEnabled: boolean;
  oidcGenericEnabled: boolean;
  // The providers configured by the organization owning this host. Only set on a custom app domain.
  oidcProviders: PortalOIDCProvider[];
}

export interface Portal {
  // Whether the request host matches a custom app domain. Used to drop Distr-specific branding (logo, website
  // links) on custom domains, even when no branding assets are configured.
  customDomain: boolean;
  pageTitle?: string;
  faviconUrl?: string;
  logoUrl?: string;
  loginConfig: PortalLoginConfig;
}
