package api

import (
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type CreateUpdateEnvironmentRequest struct {
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	SortOrder           int        `json:"sortOrder"`
	IsProduction        bool       `json:"isProduction"`
	AllowDynamicTargets bool       `json:"allowDynamicTargets"`
	RetentionPolicyID   *uuid.UUID `json:"retentionPolicyId,omitempty"`
}

func (r *CreateUpdateEnvironmentRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return validation.NewValidationFailedError("name is required")
	}
	if r.SortOrder < 0 {
		return validation.NewValidationFailedError("sortOrder must be non-negative")
	}
	return nil
}

type Environment struct {
	ID                  uuid.UUID  `json:"id"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	SortOrder           int        `json:"sortOrder"`
	IsProduction        bool       `json:"isProduction"`
	AllowDynamicTargets bool       `json:"allowDynamicTargets"`
	RetentionPolicyID   *uuid.UUID `json:"retentionPolicyId,omitempty"`
}
