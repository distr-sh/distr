package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func NotificationRecordWithCurrentStatusToAPI(
	record types.NotificationRecordWithCurrentStatus,
) api.NotificationRecordWithCurrentStatus {
	apiRecord := api.NotificationRecordWithCurrentStatus{
		NotificationRecord: api.NotificationRecord{
			ID:                 record.ID,
			CreatedAt:          record.CreatedAt,
			DeploymentTargetID: record.DeploymentTargetID,
			DeploymentStatusNotificationConfigurationID: record.DeploymentStatusNotificationConfigurationID,
			PreviousDeploymentRevisionStatusID:          record.PreviousDeploymentRevisionStatusID,
			CurrentDeploymentRevisionStatusID:           record.CurrentDeploymentRevisionStatusID,
			Message:                                     record.Message,
		},
		DeploymentTargetName:     record.DeploymentTargetName,
		CustomerOrganizationName: record.CustomerOrganizationName,
		ApplicationName:          record.ApplicationName,
		ApplicationVersionName:   record.ApplicationVersionName,
	}

	apiRecord.CurrentDeploymentRevisionStatus = PtrOrNil(
		record.CurrentDeploymentRevisionStatus,
		DeploymentRevisionStatusToAPI,
	)

	return apiRecord
}
