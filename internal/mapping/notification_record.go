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
		UpdateNotificationConfigurationID: record.UpdateNotificationConfigurationID,
		ApplicationVersionID:              record.ApplicationVersionID,
		ArtifactVersionID:                 record.ArtifactVersionID,
		ArtifactName:                      record.ArtifactName,
		ArtifactVersionName:               record.ArtifactVersionName,
		Deployments:                       List(record.Deployments, notificationRecordDeploymentToAPI),
		DeliveryError:                     record.DeliveryError,
	}
}

func notificationRecordDeploymentToAPI(deployment types.NotificationRecordDeployment) api.NotificationRecordDeployment {
	return api.NotificationRecordDeployment{
		CustomerOrganizationName: deployment.CustomerOrganizationName,
		DeploymentTargetName:     deployment.DeploymentTargetName,
		HelmReleaseName:          deployment.HelmReleaseName,
		CurrentVersionName:       deployment.CurrentVersionName,
	}
}
