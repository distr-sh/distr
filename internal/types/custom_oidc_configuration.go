package types

import (
	"time"

	"github.com/google/uuid"
)

// CustomOIDCConfiguration is an OIDC provider configured by an organization for its own users.
// It is only offered on the custom domain it is bound to, and it may only ever authenticate
// accounts that belong to its organization and to no other one (see the account exclusivity
// checks in the login flow).
type CustomOIDCConfiguration struct {
	ID                     uuid.UUID  `db:"id"                         json:"id"`
	CreatedAt              time.Time  `db:"created_at"                 json:"createdAt"`
	UpdatedAt              time.Time  `db:"updated_at"                 json:"updatedAt"`
	UpdatedByUserAccountID *uuid.UUID `db:"updated_by_user_account_id" json:"-"`
	OrganizationID         uuid.UUID  `db:"organization_id"            json:"-"`
	// CustomDomainID is the domain the provider is offered on. The provider is unreachable on
	// the instance's default host and on legacy branding domains.
	CustomDomainID uuid.UUID `db:"custom_domain_id" json:"customDomainId"`
	Name           string    `db:"name"             json:"name"`
	Enabled        bool      `db:"enabled"          json:"enabled"`
	// Issuer is canonical: it is taken from the discovery document rather than from user input.
	Issuer   string `db:"issuer"    json:"issuer"`
	ClientID string `db:"client_id" json:"clientId"`
	// ClientSecret is never serialized. The API reports whether one is stored instead.
	ClientSecret string   `db:"client_secret" json:"-"`
	Scopes       []string `db:"scopes"        json:"scopes"`
	// PKCEEnabled is nil when it should be derived from the discovery document.
	PKCEEnabled *bool `db:"pkce_enabled" json:"pkceEnabled,omitempty"`
	// SPInitiated makes the login page redirect to this provider without user interaction.
	SPInitiated bool `db:"sp_initiated" json:"spInitiated"`
	// CreateUnknownUsers provisions an account on first login for an email that has none yet,
	// subject to AllowedEmailDomains and the organization's user account limit.
	CreateUnknownUsers  bool     `db:"create_unknown_users"  json:"createUnknownUsers"`
	DefaultUserRole     UserRole `db:"default_user_role"     json:"defaultUserRole"`
	AllowedEmailDomains []string `db:"allowed_email_domains" json:"allowedEmailDomains"`
}
