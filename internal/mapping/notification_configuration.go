package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func ApplicationNotificationConfigurationToAPI(
	config types.ApplicationNotificationConfiguration,
) api.ApplicationNotificationConfiguration {
	return api.ApplicationNotificationConfiguration{
		ID:                            config.ID,
		CreatedAt:                     config.CreatedAt,
		Name:                          config.Name,
		Enabled:                       config.Enabled,
		UpdateAvailableTriggerEnabled: config.UpdateAvailableTriggerEnabled,
		Applications:                  List(config.Applications, notificationApplicationToAPI),
		Recipients:                    List(config.Recipients, notificationRecipientToAPI),
	}
}

func ApplicationNotificationConfigurationToInternal(
	request api.CreateUpdateApplicationNotificationConfigurationRequest,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) types.ApplicationNotificationConfiguration {
	return types.ApplicationNotificationConfiguration{
		OrganizationID:                organizationID,
		CustomerOrganizationID:        customerOrganizationID,
		Name:                          request.Name,
		Enabled:                       request.Enabled,
		UpdateAvailableTriggerEnabled: request.UpdateAvailableTriggerEnabled,
		ApplicationIDs:                request.ApplicationIDs,
		UserAccountIDs:                request.UserAccountIDs,
	}
}

func ArtifactNotificationConfigurationToAPI(
	config types.ArtifactNotificationConfiguration,
) api.ArtifactNotificationConfiguration {
	return api.ArtifactNotificationConfiguration{
		ID:                       config.ID,
		CreatedAt:                config.CreatedAt,
		Name:                     config.Name,
		Enabled:                  config.Enabled,
		NewVersionTriggerEnabled: config.NewVersionTriggerEnabled,
		Artifacts:                List(config.Artifacts, notificationArtifactToAPI),
		Recipients:               List(config.Recipients, notificationRecipientToAPI),
	}
}

func ArtifactNotificationConfigurationToInternal(
	request api.CreateUpdateArtifactNotificationConfigurationRequest,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) types.ArtifactNotificationConfiguration {
	return types.ArtifactNotificationConfiguration{
		OrganizationID:           organizationID,
		CustomerOrganizationID:   customerOrganizationID,
		Name:                     request.Name,
		Enabled:                  request.Enabled,
		NewVersionTriggerEnabled: request.NewVersionTriggerEnabled,
		ArtifactIDs:              request.ArtifactIDs,
		UserAccountIDs:           request.UserAccountIDs,
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
