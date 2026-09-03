package api

import "github.com/distr-sh/distr/internal/env"

type PortalResponse struct {
	CustomDomain bool              `json:"customDomain"`
	PageTitle    *string           `json:"pageTitle,omitempty"`
	FaviconUrl   *string           `json:"faviconUrl,omitempty"`
	LogoUrl      *string           `json:"logoUrl,omitempty"`
	LoginConfig  PortalLoginConfig `json:"loginConfig"`
}

type PortalLoginConfig struct {
	Registration         env.RegistrationMode `json:"registration"`
	TurnstileSiteKey     *string              `json:"turnstileSiteKey,omitempty"`
	OIDCGithubEnabled    bool                 `json:"oidcGithubEnabled"`
	OIDCGoogleEnabled    bool                 `json:"oidcGoogleEnabled"`
	OIDCMicrosoftEnabled bool                 `json:"oidcMicrosoftEnabled"`
	OIDCGenericEnabled   bool                 `json:"oidcGenericEnabled"`
	OIDCProviders        []PortalOIDCProvider `json:"oidcProviders,omitempty"`
}

type PortalOIDCProvider struct {
	Name        string `json:"name"`
	LoginPath   string `json:"loginPath"`
	SPInitiated bool   `json:"spInitiated"`
}
