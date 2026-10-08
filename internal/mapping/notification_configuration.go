package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func UpdateNotificationConfigurationToAPI(
	config types.UpdateNotificationConfiguration,
) api.UpdateNotificationConfiguration {
	return api.UpdateNotificationConfiguration{
		ID:           config.ID,
		CreatedAt:    config.CreatedAt,
		Name:         config.Name,
		Enabled:      config.Enabled,
		Applications: List(config.Applications, notificationApplicationToAPI),
		Artifacts:    List(config.Artifacts, notificationArtifactToAPI),
		Recipients:   List(config.Recipients, notificationRecipientToAPI),
	}
}

func UpdateNotificationConfigurationToInternal(
	request api.CreateUpdateNotificationConfigurationRequest,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) types.UpdateNotificationConfiguration {
	return types.UpdateNotificationConfiguration{
		OrganizationID:         organizationID,
		CustomerOrganizationID: customerOrganizationID,
		Name:                   request.Name,
		Enabled:                request.Enabled,
		ApplicationIDs:         request.ApplicationIDs,
		ArtifactIDs:            request.ArtifactIDs,
		UserAccountIDs:         request.UserAccountIDs,
	}
}

func notificationApplicationToAPI(application types.NotificationApplication) api.NotificationApplication {
	return api.NotificationApplication{
		ID:       application.ID,
		Name:     application.Name,
		Type:     application.Type,
		ImageUrl: CreateImageURL(application.ImageID),
	}
}

func notificationArtifactToAPI(artifact types.NotificationArtifact) api.NotificationArtifact {
	return api.NotificationArtifact{
		ID:       artifact.ID,
		Name:     artifact.Name,
		ImageUrl: CreateImageURL(artifact.ImageID),
	}
}

func notificationRecipientToAPI(recipient types.NotificationRecipient) api.NotificationRecipient {
	return api.NotificationRecipient{
		ID:                     recipient.ID,
		Email:                  recipient.Email,
		Name:                   recipient.Name,
		CustomerOrganizationID: recipient.CustomerOrganizationID,
		PartnerOrganizationID:  recipient.PartnerOrganizationID,
	}
}
