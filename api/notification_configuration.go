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

type ApplicationNotificationConfiguration struct {
	ID                            uuid.UUID                 `json:"id"`
	CreatedAt                     time.Time                 `json:"createdAt"`
	Name                          string                    `json:"name"`
	Enabled                       bool                      `json:"enabled"`
	UpdateAvailableTriggerEnabled bool                      `json:"updateAvailableTriggerEnabled"`
	CustomerMessage               *string                   `json:"customerMessage,omitempty"`
	Applications                  []NotificationApplication `json:"applications"`
	Recipients                    []NotificationRecipient   `json:"recipients"`
}

type CreateUpdateApplicationNotificationConfigurationRequest struct {
	Name                          string      `json:"name"`
	Enabled                       bool        `json:"enabled"`
	UpdateAvailableTriggerEnabled bool        `json:"updateAvailableTriggerEnabled"`
	CustomerMessage               *string     `json:"customerMessage,omitempty"`
	ApplicationIDs                []uuid.UUID `json:"applicationIds"`
	UserAccountIDs                []uuid.UUID `json:"userAccountIds"`
}

func (r CreateUpdateApplicationNotificationConfigurationRequest) Validate() error {
	if err := validateNotificationConfiguration(r.Name, len(r.ApplicationIDs), len(r.UserAccountIDs),
		"application"); err != nil {
		return err
	}
	if !r.UpdateAvailableTriggerEnabled {
		return validation.NewValidationFailedError("enable at least one trigger")
	}
	return nil
}

type ArtifactNotificationConfiguration struct {
	ID                       uuid.UUID               `json:"id"`
	CreatedAt                time.Time               `json:"createdAt"`
	Name                     string                  `json:"name"`
	Enabled                  bool                    `json:"enabled"`
	NewVersionTriggerEnabled bool                    `json:"newVersionTriggerEnabled"`
	CustomerMessage          *string                 `json:"customerMessage,omitempty"`
	Artifacts                []NotificationArtifact  `json:"artifacts"`
	Recipients               []NotificationRecipient `json:"recipients"`
}

type CreateUpdateArtifactNotificationConfigurationRequest struct {
	Name                     string      `json:"name"`
	Enabled                  bool        `json:"enabled"`
	NewVersionTriggerEnabled bool        `json:"newVersionTriggerEnabled"`
	CustomerMessage          *string     `json:"customerMessage,omitempty"`
	ArtifactIDs              []uuid.UUID `json:"artifactIds"`
	UserAccountIDs           []uuid.UUID `json:"userAccountIds"`
}

func (r CreateUpdateArtifactNotificationConfigurationRequest) Validate() error {
	if err := validateNotificationConfiguration(r.Name, len(r.ArtifactIDs), len(r.UserAccountIDs),
		"artifact"); err != nil {
		return err
	}
	if !r.NewVersionTriggerEnabled {
		return validation.NewValidationFailedError("enable at least one trigger")
	}
	return nil
}

func validateNotificationConfiguration(name string, subjects, recipients int, subjectLabel string) error {
	if name == "" {
		return validation.NewValidationFailedError("a name is required")
	}
	if subjects == 0 {
		return validation.NewValidationFailedError("select at least one " + subjectLabel)
	}
	if recipients == 0 {
		return validation.NewValidationFailedError("select at least one recipient")
	}
	return nil
}
