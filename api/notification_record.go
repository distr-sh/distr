package api

import (
	"time"

	"github.com/google/uuid"
)

type NotificationRecord struct {
	ID                                uuid.UUID                `json:"id"`
	CreatedAt                         time.Time                `json:"createdAt"`
	DeploymentTargetID                *uuid.UUID               `json:"deploymentTargetId"`
	DeploymentTargetName              *string                  `json:"deploymentTargetName,omitempty"`
	CustomerOrganizationName          *string                  `json:"customerOrganizationName,omitempty"`
	AlertConfigurationID              *uuid.UUID               `json:"alertConfigurationId,omitempty"`
	Type                              string                   `json:"type"`
	DeploymentRevisionID              *uuid.UUID               `json:"deploymentRevisionId,omitempty"`
	ApplicationName                   *string                  `json:"applicationName,omitempty"`
	ApplicationVersionName            *string                  `json:"applicationVersionName,omitempty"`
	DeploymentStatusMessage           *string                  `json:"deploymentStatusMessage,omitempty"`
	MetricType                        *string                  `json:"metricType,omitempty"`
	DiskDevice                        *string                  `json:"diskDevice,omitempty"`
	DiskPath                          *string                  `json:"diskPath,omitempty"`
	PreviousDeploymentTargetMetricsID *uuid.UUID               `json:"previousDeploymentTargetMetricsId,omitempty"`
	CurrentDeploymentTargetMetricsID  *uuid.UUID               `json:"currentDeploymentTargetMetricsId,omitempty"`
	CurrentDeploymentTargetMetrics    *DeploymentTargetMetrics `json:"currentDeploymentTargetMetrics,omitempty"`
	DeliveryError                     string                   `json:"deliveryError"`
}
