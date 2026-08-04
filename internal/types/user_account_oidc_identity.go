package types

import (
	"time"

	"github.com/google/uuid"
)

type OIDCProvider string

const (
	OIDCProviderGithub    OIDCProvider = "github"
	OIDCProviderGoogle    OIDCProvider = "google"
	OIDCProviderMicrosoft OIDCProvider = "microsoft"
	OIDCProviderGeneric   OIDCProvider = "generic"
	// OIDCProviderCustom is an identity from a CustomOIDCConfiguration, i.e. from a provider
	// configured by an organization rather than by the instance.
	OIDCProviderCustom OIDCProvider = "custom"
)

// UserAccountOIDCIdentity is the identity of a user account at an identity provider.
// Logins are matched on (Issuer, Subject), which stays stable when the email address
// changes at the provider or in Distr.
type UserAccountOIDCIdentity struct {
	ID            uuid.UUID    `db:"id"              json:"id"`
	CreatedAt     time.Time    `db:"created_at"      json:"createdAt"`
	UserAccountID uuid.UUID    `db:"user_account_id" json:"-"`
	Provider      OIDCProvider `db:"provider"        json:"provider"`
	Issuer        string       `db:"issuer"          json:"issuer"`
	Subject       string       `db:"subject"         json:"-"`
	// Email as reported by the identity provider, for display only. The email of the
	// user account is never overwritten with it.
	Email       *string    `db:"email"         json:"email,omitempty"`
	LastLoginAt *time.Time `db:"last_login_at" json:"lastLoginAt,omitempty"`
	// CustomOIDCConfigurationID is set exactly for Provider OIDCProviderCustom and names the
	// organization configuration that governs this login.
	CustomOIDCConfigurationID *uuid.UUID `db:"custom_oidc_configuration_id" json:"-"`
}

// UserAccountOIDCIdentityWithConfiguration adds the display name of the organization
// configuration behind a custom identity and the name of the organization that controls it.
// Both are nil for the instance-scoped providers.
type UserAccountOIDCIdentityWithConfiguration struct {
	UserAccountOIDCIdentity
	ConfigurationName *string `db:"configuration_name"`
	OrganizationName  *string `db:"organization_name"`
}
