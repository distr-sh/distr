package api

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

const (
	customOIDCConfigurationNameMaxLength = 100
	// openIDScope is required for an OIDC flow, so it is added rather than demanded.
	openIDScope = "openid"
)

// CustomOIDCConfiguration is an organization's own OIDC provider. The client secret is never
// returned; ClientSecretSet reports whether one is stored.
type CustomOIDCConfiguration struct {
	ID                  uuid.UUID      `json:"id"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	CustomDomainID      uuid.UUID      `json:"customDomainId"`
	Name                string         `json:"name"`
	Enabled             bool           `json:"enabled"`
	Issuer              string         `json:"issuer"`
	ClientID            string         `json:"clientId"`
	ClientSecretSet     bool           `json:"clientSecretSet"`
	Scopes              []string       `json:"scopes"`
	PKCEEnabled         *bool          `json:"pkceEnabled,omitempty"`
	SPInitiated         bool           `json:"spInitiated"`
	CreateUnknownUsers  bool           `json:"createUnknownUsers"`
	DefaultUserRole     types.UserRole `json:"defaultUserRole"`
	AllowedEmailDomains []string       `json:"allowedEmailDomains"`
	// CallbackURL is the redirect URI that has to be registered with the identity provider.
	CallbackURL string `json:"callbackUrl"`
}

// CustomOIDCConfigurationsResponse lists an organization's providers together with the number of
// members that cannot use any of them, because they are also members of another organization.
type CustomOIDCConfigurationsResponse struct {
	Configurations []CustomOIDCConfiguration `json:"configurations"`
	// MembersWithOtherOrganizations is reported rather than enforced: existing members keep their
	// membership and their password login, they just cannot sign in through the provider.
	MembersWithOtherOrganizations int64 `json:"membersWithOtherOrganizations"`
}

type CustomOIDCConfigurationRequest struct {
	CustomDomainID uuid.UUID `json:"customDomainId"`
	Name           string    `json:"name"`
	Enabled        bool      `json:"enabled"`
	Issuer         string    `json:"issuer"`
	ClientID       string    `json:"clientId"`
	// ClientSecret is write-only. An omitted value keeps the stored secret, which is how the form
	// can be saved again without the secret ever being sent to the browser.
	ClientSecret        *string        `json:"clientSecret,omitempty"`
	Scopes              []string       `json:"scopes"`
	PKCEEnabled         *bool          `json:"pkceEnabled,omitempty"`
	SPInitiated         bool           `json:"spInitiated"`
	CreateUnknownUsers  bool           `json:"createUnknownUsers"`
	DefaultUserRole     types.UserRole `json:"defaultUserRole"`
	AllowedEmailDomains []string       `json:"allowedEmailDomains"`
}

// Normalize brings the request into the form it is stored in: a trimmed name and issuer, the
// openid scope present exactly once, and email domains as bare lowercase hostnames.
func (r *CustomOIDCConfigurationRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Issuer = strings.TrimSpace(r.Issuer)
	r.ClientID = strings.TrimSpace(r.ClientID)

	scopes := []string{openIDScope}
	for _, scope := range r.Scopes {
		if scope = strings.TrimSpace(scope); scope != "" && !slices.Contains(scopes, scope) {
			scopes = append(scopes, scope)
		}
	}
	r.Scopes = scopes

	domains := make([]string, 0, len(r.AllowedEmailDomains))
	for _, domain := range r.AllowedEmailDomains {
		domain = validation.NormalizeHostname(strings.TrimPrefix(strings.TrimSpace(domain), "@"))
		if domain != "" && !slices.Contains(domains, domain) {
			domains = append(domains, domain)
		}
	}
	r.AllowedEmailDomains = domains
}

func (r *CustomOIDCConfigurationRequest) Validate() error {
	if r.CustomDomainID == uuid.Nil {
		return validation.NewValidationFailedError("customDomainId is required")
	}
	if r.Name == "" {
		return validation.NewValidationFailedError("name is required")
	}
	if len(r.Name) > customOIDCConfigurationNameMaxLength {
		return validation.NewValidationFailedError(
			fmt.Sprintf("name must be at most %v characters", customOIDCConfigurationNameMaxLength))
	}
	if _, err := oidc.ParseIssuerURL(r.Issuer); err != nil {
		return validation.NewValidationFailedError(err.Error())
	}
	if r.ClientID == "" {
		return validation.NewValidationFailedError("clientId is required")
	}
	if r.ClientSecret != nil && strings.TrimSpace(*r.ClientSecret) == "" {
		return validation.NewValidationFailedError(
			"clientSecret must not be empty. Omit it to keep the stored secret")
	}
	if _, err := types.ParseUserRole(string(r.DefaultUserRole)); err != nil {
		return validation.NewValidationFailedError("defaultUserRole is invalid")
	}
	for _, domain := range r.AllowedEmailDomains {
		if err := validation.ValidateHostname(domain); err != nil {
			return validation.NewValidationFailedError(
				fmt.Sprintf("allowedEmailDomains contains the invalid domain %q", domain))
		}
	}
	// An account created on first sign-in joins the organization's own team, so provisioning without
	// a domain restriction would hand a membership to everybody the provider can authenticate —
	// including the customer users who reach the same domain through the customer portal.
	if r.CreateUnknownUsers && len(r.AllowedEmailDomains) == 0 {
		return validation.NewValidationFailedError(
			"allowedEmailDomains is required when accounts are created on first sign-in")
	}
	return nil
}
