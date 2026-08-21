export interface PortalOIDCProvider {
  name: string;
  loginPath: string;
  spInitiated: boolean;
}

export interface PortalLoginConfig {
  registrationEnabled: boolean;
  turnstileSiteKey?: string;
  oidcGithubEnabled: boolean;
  oidcGoogleEnabled: boolean;
  oidcMicrosoftEnabled: boolean;
  oidcGenericEnabled: boolean;
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
