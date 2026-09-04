package types

import (
	"time"

	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
)

type CustomOIDCConfiguration struct {
	ID                     uuid.UUID       `db:"id"`
	CreatedAt              time.Time       `db:"created_at"`
	UpdatedAt              time.Time       `db:"updated_at"`
	UpdatedByUserAccountID *uuid.UUID      `db:"updated_by_user_account_id"`
	OrganizationID         uuid.UUID       `db:"organization_id"`
	CustomDomainID         uuid.UUID       `db:"custom_domain_id"`
	Name                   string          `db:"name"`
	Slug                   string          `db:"slug"`
	Enabled                bool            `db:"enabled"`
	Issuer                 string          `db:"issuer"`
	ClientID               string          `db:"client_id"`
	ClientSecret           dbcrypto.String `db:"client_secret"`
	Scopes                 []string        `db:"scopes"`
	PKCEEnabled            *bool           `db:"pkce_enabled"`
	SPInitiated            bool            `db:"sp_initiated"`
	CreateUnknownUsers     bool            `db:"create_unknown_users"`
	DefaultUserRole        UserRole        `db:"default_user_role"`
	AllowedEmailDomains    []string        `db:"allowed_email_domains"`
}
