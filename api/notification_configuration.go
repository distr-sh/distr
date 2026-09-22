package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type NotificationRecipient struct {
	ID                     uuid.UUID  `json:"id"`
	Email                  string     `json:"email"`
	Name                   string     `json:"name,omitempty"`
	CustomerOrganizationID *uuid.UUID `json:"customerOrganizationId,omitempty"`
	PartnerOrganizationID  *uuid.UUID `json:"partnerOrganizationId,omitempty"`
}

type NotificationApplication struct {
	ID       uuid.UUID            `json:"id"`
	Name     string               `json:"name"`
	Type     types.DeploymentType `json:"type"`
	ImageUrl *string              `json:"imageUrl,omitempty"`
}

type NotificationArtifact struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	ImageUrl *string   `json:"imageUrl,omitempty"`
}

type UpdateNotificationConfiguration struct {
	ID           uuid.UUID                 `json:"id"`
	CreatedAt    time.Time                 `json:"createdAt"`
	Name         string                    `json:"name"`
	Enabled      bool                      `json:"enabled"`
	Applications []NotificationApplication `json:"applications"`
	Artifacts    []NotificationArtifact    `json:"artifacts"`
	Recipients   []NotificationRecipient   `json:"recipients"`
}

type CreateUpdateNotificationConfigurationRequest struct {
	CustomerOrganizationID *uuid.UUID  `json:"customerOrganizationId,omitempty"`
	Name                   string      `json:"name"`
	Enabled                bool        `json:"enabled"`
	ApplicationIDs         []uuid.UUID `json:"applicationIds"`
	ArtifactIDs            []uuid.UUID `json:"artifactIds"`
	UserAccountIDs         []uuid.UUID `json:"userAccountIds"`
}

func (r CreateUpdateNotificationConfigurationRequest) Validate() error {
	if r.Name == "" {
		return validation.NewValidationFailedError("a name is required")
	}
	if len(r.ApplicationIDs) == 0 && len(r.ArtifactIDs) == 0 {
		return validation.NewValidationFailedError("select at least one application or artifact")
	}
	if len(r.UserAccountIDs) == 0 {
		return validation.NewValidationFailedError("select at least one recipient")
	}
	return nil
}
