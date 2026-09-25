package types

import (
	"time"

	"github.com/google/uuid"
)

type NotificationRecordType string

const (
	NotificationRecordTypeAlert    NotificationRecordType = "alert"
	NotificationRecordTypeWarning  NotificationRecordType = "warning"
	NotificationRecordTypeResolved NotificationRecordType = "resolved"
)

type NotificationRecord struct {
	ID                     uuid.UUID              `db:"id"`
	CreatedAt              time.Time              `db:"created_at"`
	OrganizationID         uuid.UUID              `db:"organization_id"`
	CustomerOrganizationID *uuid.UUID             `db:"customer_organization_id"`
	DeploymentTargetID     *uuid.UUID             `db:"deployment_target_id"`
	AlertConfigurationID   *uuid.UUID             `db:"alert_configuration_id"`
	Type                   NotificationRecordType `db:"type"`
	DeploymentRevisionID   *uuid.UUID             `db:"deployment_revision_id"`
	// DeploymentStatusMessage is a copy of the status message at the time of the notification, since the revision
	// only keeps its latest status. Stale warnings have none.
	DeploymentStatusMessage *string `db:"deployment_status_message"`
	// ResolvedAt is set on a stale warning once the agent reports again. An unresolved warning suppresses further
	// warnings for the same deployment and alert configuration.
	ResolvedAt                        *time.Time `db:"resolved_at"`
	MetricType                        *string    `db:"metric_type"`
	DiskDevice                        *string    `db:"disk_device"`
	DiskPath                          *string    `db:"disk_path"`
	PreviousDeploymentTargetMetricsID *uuid.UUID `db:"previous_deployment_target_metrics_id"`
	CurrentDeploymentTargetMetricsID  *uuid.UUID `db:"current_deployment_target_metrics_id"`
	DeliveryError                     string     `db:"delivery_error"`
}

type NotificationRecordWithDetails struct {
	NotificationRecord
	DeploymentTargetName           *string                  `db:"deployment_target_name"`
	CustomerOrganizationName       *string                  `db:"customer_organization_name"`
	ApplicationName                *string                  `db:"application_name"`
	ApplicationVersionName         *string                  `db:"application_version_name"`
	CurrentDeploymentTargetMetrics *DeploymentTargetMetrics `db:"current_deployment_target_metrics"`
}
