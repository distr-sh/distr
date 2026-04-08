package api

import (
	"encoding/json"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type CreateLicenseKeyRequest struct {
	Name                   string          `json:"name"`
	Description            *string         `json:"description,omitempty"`
	Payload                json.RawMessage `json:"payload"`
	NotBefore              time.Time       `json:"notBefore"`
	ExpiresAt              time.Time       `json:"expiresAt"`
	CustomerOrganizationID *uuid.UUID      `json:"customerOrganizationId,omitempty"`
}

type UpdateLicenseKeyRequest struct {
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	NotBefore   *time.Time       `json:"notBefore,omitempty"`
	ExpiresAt   *time.Time       `json:"expiresAt,omitempty"`
	Payload     *json.RawMessage `json:"payload,omitempty"`
}

type LicenseKeyRevision struct {
	types.LicenseKeyRevision

	Token string `json:"token"`
}
