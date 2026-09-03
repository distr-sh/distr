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
	OIDCProviderCustom    OIDCProvider = "custom"
)

// UserAccountOIDCIdentity is the identity of a user account at an identity provider.
// Logins are matched on (Issuer, Subject), which stays stable when the email address
// changes at the provider or in Distr.
type UserAccountOIDCIdentity struct {
	ID            uuid.UUID    `db:"id"`
	CreatedAt     time.Time    `db:"created_at"`
	UserAccountID uuid.UUID    `db:"user_account_id"`
	Provider      OIDCProvider `db:"provider"`
	Issuer        string       `db:"issuer"`
	Subject       string       `db:"subject"`
	// Email as reported by the identity provider, for display only. The email of the
	// user account is never overwritten with it.
	Email                     *string    `db:"email"`
	LastLoginAt               *time.Time `db:"last_login_at"`
	CustomOIDCConfigurationID *uuid.UUID `db:"custom_oidc_configuration_id"`
}

type UserAccountOIDCIdentityWithConfiguration struct {
	UserAccountOIDCIdentity
	ConfigurationName *string `db:"configuration_name"`
	OrganizationName  *string `db:"organization_name"`
}
