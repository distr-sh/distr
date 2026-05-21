package types

import (
	"time"

	"github.com/distr-sh/distr/internal/authkey"
	"github.com/google/uuid"
)

type ServiceAccount struct {
	ID                     uuid.UUID   `db:"id"`
	CreatedAt              time.Time   `db:"created_at"`
	OrganizationID         uuid.UUID   `db:"organization_id"`
	CustomerOrganizationID *uuid.UUID  `db:"customer_organization_id"`
	Name                   string      `db:"name"`
	AccountRole            AccountRole `db:"account_role"`
}

type ServiceAccountAccessToken struct {
	ID               uuid.UUID   `db:"id"`
	CreatedAt        time.Time   `db:"created_at"`
	ExpiresAt        *time.Time  `db:"expires_at"`
	LastUsedAt       *time.Time  `db:"last_used_at"`
	Label            *string     `db:"label"`
	Key              authkey.Key `db:"key"`
	ServiceAccountID uuid.UUID   `db:"service_account_id"`
}

type ServiceAccountAccessTokenWithServiceAccount struct {
	ServiceAccountAccessToken
	ServiceAccount ServiceAccount `db:"service_account"`
}
