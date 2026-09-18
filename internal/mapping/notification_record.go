package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func NotificationRecordToAPI(record types.NotificationRecord) api.NotificationRecord {
	return api.NotificationRecord{
		ID:                    record.ID,
		CreatedAt:             record.CreatedAt,
		UserAccountID:         record.UserAccountID,
		SourceType:            record.SourceType,
		SourceConfigurationID: record.SourceConfigurationID,
		SubjectID:             record.SubjectID,
		Type:                  record.Type,
		Details:               notificationRecordDetailsToAPI(record.Details),
		Message:               record.Message,
	}
}

// notificationRecordDetailsToAPI drops the identifiers that only the notification logic itself
// needs, so that the response carries what the history displays and nothing else.
func notificationRecordDetailsToAPI(details types.NotificationRecordDetails) api.NotificationRecordDetails {
	return api.NotificationRecordDetails{
		Summary:                  details.Summary,
		CustomerOrganizationName: details.CustomerOrganizationName,
		DeploymentTargetName:     details.DeploymentTargetName,
		ApplicationName:          details.ApplicationName,
		ApplicationType:          details.ApplicationType,
		ApplicationVersionName:   details.ApplicationVersionName,
		ArtifactName:             details.ArtifactName,
		ArtifactVersionName:      details.ArtifactVersionName,
		Deployments:              List(details.Deployments, notificationRecordDeploymentToAPI),
	}
}

func notificationRecordDeploymentToAPI(
	deployment types.NotificationRecordDeployment,
) api.NotificationRecordDeployment {
	return api.NotificationRecordDeployment{
		CustomerOrganizationName: deployment.CustomerOrganizationName,
		DeploymentTargetName:     deployment.DeploymentTargetName,
		DeploymentName:           deployment.DeploymentName,
		CurrentVersionName:       deployment.CurrentVersionName,
	}
}
