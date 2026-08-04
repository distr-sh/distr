package api

import "github.com/google/uuid"

// PortalResponse is the host-resolved bootstrap configuration for the unauthenticated pages: the portal branding
// (browser tab title, favicon and logo) that applies to everyone visiting an organization's custom app domain, plus
// the login methods available on this host. The branding fields are empty unless the host matches a custom app
// domain whose organization has configured branding.
type PortalResponse struct {
	// CustomDomain reports whether the request host resolves to an organization's custom app domain, which is not
	// the same as having branding: a domain can be registered long before any branding is saved.
	CustomDomain bool              `json:"customDomain"`
	PageTitle    *string           `json:"pageTitle,omitempty"`
	FaviconUrl   *string           `json:"faviconUrl,omitempty"`
	LogoUrl      *string           `json:"logoUrl,omitempty"`
	LoginConfig  PortalLoginConfig `json:"loginConfig"`
}

// PortalLoginConfig describes the login methods offered on the request host. The instance-scoped OIDC providers
// are configured per instance via environment variables, but are only offered on the default host and on legacy
// branding app domains -- never on a self-service custom domain, where only that organization's own providers apply.
// Registration follows the same rule: signing up creates a new organization on the platform, which is not something
// a vendor's domain offers.
type PortalLoginConfig struct {
	RegistrationEnabled  bool `json:"registrationEnabled"`
	OIDCGithubEnabled    bool `json:"oidcGithubEnabled"`
	OIDCGoogleEnabled    bool `json:"oidcGoogleEnabled"`
	OIDCMicrosoftEnabled bool `json:"oidcMicrosoftEnabled"`
	OIDCGenericEnabled   bool `json:"oidcGenericEnabled"`
	// OIDCProviders are the providers configured by the organization owning the request host.
	OIDCProviders []PortalOIDCProvider `json:"oidcProviders,omitempty"`
}

// PortalOIDCProvider is one of an organization's own OIDC providers, as offered on its custom domain. Only what the
// login page needs is exposed: the login is started at /api/v1/auth/oidc/custom/{id}.
type PortalOIDCProvider struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	// SPInitiated makes the login page redirect to this provider without user interaction.
	SPInitiated bool `json:"spInitiated"`
}
