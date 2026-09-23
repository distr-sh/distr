package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func NotificationRecordWithCurrentStatusToAPI(
	record types.NotificationRecordWithCurrentStatus,
) api.NotificationRecordWithCurrentStatus {
	return api.NotificationRecordWithCurrentStatus{
		NotificationRecord: api.NotificationRecord{
			ID:                                record.ID,
			CreatedAt:                         record.CreatedAt,
			DeploymentTargetID:                record.DeploymentTargetID,
			AlertConfigurationID:              record.AlertConfigurationID,
			Type:                              string(record.Type),
			PreviousDeploymentRevisionID:      record.PreviousDeploymentRevisionID,
			CurrentDeploymentRevisionID:       record.CurrentDeploymentRevisionID,
			MetricType:                        record.MetricType,
			DiskDevice:                        record.DiskDevice,
			DiskPath:                          record.DiskPath,
			PreviousDeploymentTargetMetricsID: record.PreviousDeploymentTargetMetricsID,
			CurrentDeploymentTargetMetricsID:  record.CurrentDeploymentTargetMetricsID,
			Message:                           record.Message,
		},
		DeploymentTargetName:     record.DeploymentTargetName,
		CustomerOrganizationName: record.CustomerOrganizationName,
		ApplicationName:          record.ApplicationName,
		ApplicationVersionName:   record.ApplicationVersionName,
		CurrentDeploymentRevisionStatus: PtrOrNil(
			record.CurrentStatus(),
			DeploymentRevisionStatusToAPI,
		),
		CurrentDeploymentTargetMetrics: PtrOrNil(
			record.CurrentDeploymentTargetMetrics,
			DeploymentTargetMetricsToAPI,
		),
	}
}
