package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func NotificationRecordWithDetailsToAPI(record types.NotificationRecordWithDetails) api.NotificationRecord {
	return api.NotificationRecord{
		ID:                                record.ID,
		CreatedAt:                         record.CreatedAt,
		DeploymentTargetID:                record.DeploymentTargetID,
		DeploymentTargetName:              record.DeploymentTargetName,
		CustomerOrganizationName:          record.CustomerOrganizationName,
		AlertConfigurationID:              record.AlertConfigurationID,
		Type:                              string(record.Type),
		DeploymentRevisionID:              record.DeploymentRevisionID,
		ApplicationName:                   record.ApplicationName,
		ApplicationVersionName:            record.ApplicationVersionName,
		DeploymentStatusMessage:           record.DeploymentStatusMessage,
		MetricType:                        record.MetricType,
		DiskDevice:                        record.DiskDevice,
		DiskPath:                          record.DiskPath,
		PreviousDeploymentTargetMetricsID: record.PreviousDeploymentTargetMetricsID,
		CurrentDeploymentTargetMetricsID:  record.CurrentDeploymentTargetMetricsID,
		CurrentDeploymentTargetMetrics: PtrOrNil(
			record.CurrentDeploymentTargetMetrics,
			DeploymentTargetMetricsToAPI,
		),
		DeliveryError: record.DeliveryError,
	}
}
