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
	Key                    string     `json:"key"`
	Value                  string     `json:"value"`
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
}

type UpdateSecretRequest struct {
	ID     uuid.UUID `path:"secretId"`
	Value  string    `json:"value"`
	DryRun bool      `json:"dryRun"`
}

type DeleteSecretRequest struct {
	ID uuid.UUID `path:"secretId"`
}

type UpdateSecretResponse struct {
	Secret              SecretWithoutValue   `json:"secret"`
	AffectedDeployments []AffectedDeployment `json:"affectedDeployments"`
}
