package api

import (
	"time"

	"github.com/google/uuid"
)

type NotificationRecord struct {
	ID                                uuid.UUID                      `json:"id"`
	CreatedAt                         time.Time                      `json:"createdAt"`
	DeploymentTargetID                *uuid.UUID                     `json:"deploymentTargetId"`
	DeploymentTargetName              *string                        `json:"deploymentTargetName,omitempty"`
	CustomerOrganizationName          *string                        `json:"customerOrganizationName,omitempty"`
	AlertConfigurationID              *uuid.UUID                     `json:"alertConfigurationId,omitempty"`
	Type                              string                         `json:"type"`
	DeploymentRevisionID              *uuid.UUID                     `json:"deploymentRevisionId,omitempty"`
	ApplicationName                   *string                        `json:"applicationName,omitempty"`
	ApplicationVersionName            *string                        `json:"applicationVersionName,omitempty"`
	DeploymentStatusMessage           *string                        `json:"deploymentStatusMessage,omitempty"`
	MetricType                        *string                        `json:"metricType,omitempty"`
	DiskDevice                        *string                        `json:"diskDevice,omitempty"`
	DiskPath                          *string                        `json:"diskPath,omitempty"`
	PreviousDeploymentTargetMetricsID *uuid.UUID                     `json:"previousDeploymentTargetMetricsId,omitempty"`
	CurrentDeploymentTargetMetricsID  *uuid.UUID                     `json:"currentDeploymentTargetMetricsId,omitempty"`
	CurrentDeploymentTargetMetrics    *DeploymentTargetMetrics       `json:"currentDeploymentTargetMetrics,omitempty"`
	UpdateNotificationConfigurationID *uuid.UUID                     `json:"updateNotificationConfigurationId,omitempty"`
	ApplicationVersionID              *uuid.UUID                     `json:"applicationVersionId,omitempty"`
	ArtifactVersionID                 *uuid.UUID                     `json:"artifactVersionId,omitempty"`
	ArtifactName                      *string                        `json:"artifactName,omitempty"`
	ArtifactVersionName               *string                        `json:"artifactVersionName,omitempty"`
	Deployments                       []NotificationRecordDeployment `json:"deployments,omitempty"`
	DeliveryError                     string                         `json:"deliveryError"`
}

type NotificationRecordDeployment struct {
	CustomerOrganizationName *string `json:"customerOrganizationName,omitempty"`
	DeploymentTargetName     string  `json:"deploymentTargetName"`
	HelmReleaseName          *string `json:"helmReleaseName,omitempty"`
	CurrentVersionName       string  `json:"currentVersionName"`
}
