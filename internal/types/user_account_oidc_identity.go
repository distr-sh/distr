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
}
