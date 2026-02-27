package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type UsageLicense struct {
	ID                     uuid.UUID       `db:"id" json:"id"`
	CreatedAt              time.Time       `db:"created_at" json:"createdAt"`
	Name                   string          `db:"name" json:"name"`
	Description            *string         `db:"description" json:"description,omitempty"`
	Payload                json.RawMessage `db:"payload" json:"payload"`
	Token                  string          `db:"token" json:"token"`
	NotBefore              time.Time       `db:"not_before" json:"notBefore"`
	ExpiresAt              time.Time       `db:"expires_at" json:"expiresAt"`
	OrganizationID         uuid.UUID       `db:"organization_id" json:"-"`
	CustomerOrganizationID *uuid.UUID      `db:"customer_organization_id" json:"customerOrganizationId,omitempty"`
}
