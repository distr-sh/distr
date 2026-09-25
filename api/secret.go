package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type SecretWithoutValue struct {
	ID                     uuid.UUID          `json:"id"`
	CreatedAt              time.Time          `json:"createdAt"`
	UpdatedAt              time.Time          `json:"updatedAt"`
	UpdatedBy              *types.UserAccount `json:"updatedBy,omitempty"`
	CustomerOrganizationID *uuid.UUID         `json:"customerOrganizationId,omitempty"`
	Key                    string             `json:"key"`
}

type CreateSecretRequest struct {
	Key string `json:"key"`
	// A secret value is stored and injected verbatim, so its surrounding whitespace is significant:
	// a certificate or private key ends in a newline.
	Value                  string     `json:"value" trim:"-"`
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
}

type UpdateSecretRequest struct {
	ID      uuid.UUID `path:"secretId"`
	Confirm bool      `query:"confirm"`
	Value   string    `json:"value" trim:"-"`
}

type DeleteSecretRequest struct {
	ID uuid.UUID `path:"secretId"`
}

type UpdateSecretResponse struct {
	SecretWithoutValue
	AffectedDeployments []AffectedDeployment `json:"affectedDeployments"`
}
