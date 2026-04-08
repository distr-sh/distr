package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type LicenseKey struct {
	ID                     uuid.UUID       `db:"id" json:"id"`
	CreatedAt              time.Time       `db:"created_at" json:"createdAt"`
	Name                   string          `db:"name" json:"name"`
	Description            *string         `db:"description" json:"description,omitempty"`
	Payload                json.RawMessage `db:"payload" json:"payload"`
	NotBefore              time.Time       `db:"not_before" json:"notBefore"`
	ExpiresAt              time.Time       `db:"expires_at" json:"expiresAt"`
	LastRevisedAt          time.Time       `db:"last_revised_at" json:"lastRevisedAt"`
	OrganizationID         uuid.UUID       `db:"organization_id" json:"-"`
	CustomerOrganizationID *uuid.UUID      `db:"customer_organization_id" json:"customerOrganizationId,omitempty"`
}

type LicenseKeyRevision struct {
	ID           uuid.UUID       `db:"id" json:"id"`
	CreatedAt    time.Time       `db:"created_at" json:"createdAt"`
	LicenseKeyID uuid.UUID       `db:"license_key_id" json:"-"`
	NotBefore    time.Time       `db:"not_before" json:"notBefore"`
	ExpiresAt    time.Time       `db:"expires_at" json:"expiresAt"`
	Payload      json.RawMessage `db:"payload" json:"payload"`
}
