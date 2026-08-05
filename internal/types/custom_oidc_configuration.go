package types

import (
	"time"

	"github.com/google/uuid"
)

type CustomOIDCConfiguration struct {
	ID                     uuid.UUID  `db:"id"                         json:"id"`
	CreatedAt              time.Time  `db:"created_at"                 json:"createdAt"`
	UpdatedAt              time.Time  `db:"updated_at"                 json:"updatedAt"`
	UpdatedByUserAccountID *uuid.UUID `db:"updated_by_user_account_id" json:"-"`
	OrganizationID         uuid.UUID  `db:"organization_id"            json:"-"`
	CustomDomainID         uuid.UUID  `db:"custom_domain_id" json:"customDomainId"`
	Name                   string     `db:"name"             json:"name"`
	Slug                   string     `db:"slug"             json:"slug"`
	Enabled                bool       `db:"enabled"          json:"enabled"`
	Issuer                 string     `db:"issuer"    json:"issuer"`
	ClientID               string     `db:"client_id" json:"clientId"`
	ClientSecret           string     `db:"client_secret" json:"-"`
	Scopes                 []string   `db:"scopes"        json:"scopes"`
	PKCEEnabled            *bool      `db:"pkce_enabled" json:"pkceEnabled,omitempty"`
	SPInitiated            bool       `db:"sp_initiated" json:"spInitiated"`
	CreateUnknownUsers     bool       `db:"create_unknown_users"  json:"createUnknownUsers"`
	DefaultUserRole        UserRole   `db:"default_user_role"     json:"defaultUserRole"`
	AllowedEmailDomains    []string   `db:"allowed_email_domains" json:"allowedEmailDomains"`
}
