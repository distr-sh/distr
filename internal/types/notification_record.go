package types

import (
	"time"

	"github.com/google/uuid"
)

type NotificationRecordType string

const (
	NotificationRecordTypeAlert           NotificationRecordType = "alert"
	NotificationRecordTypeWarning         NotificationRecordType = "warning"
	NotificationRecordTypeResolved        NotificationRecordType = "resolved"
	NotificationRecordTypeUpdateAvailable NotificationRecordType = "update_available"
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
	// ResolvedAt is set on a stale warning once the controller reports again. An unresolved warning suppresses further
	// warnings for the same deployment and alert configuration.
	ResolvedAt                        *time.Time `db:"resolved_at"`
	MetricType                        *string    `db:"metric_type"`
	DiskDevice                        *string    `db:"disk_device"`
	DiskPath                          *string    `db:"disk_path"`
	PreviousDeploymentTargetMetricsID *uuid.UUID `db:"previous_deployment_target_metrics_id"`
	CurrentDeploymentTargetMetricsID  *uuid.UUID `db:"current_deployment_target_metrics_id"`

	// UpdateNotificationConfigurationID is set on the record of an update notification, which is about exactly one
	// of ApplicationVersionID and ArtifactVersionID. Such a record is written once per audience: CustomerOrganizationID
	// and PartnerOrganizationID name the customer or partner whose users were notified, and are both nil for the
	// vendor's own team.
	UpdateNotificationConfigurationID *uuid.UUID `db:"update_notification_configuration_id"`
	ApplicationVersionID              *uuid.UUID `db:"application_version_id"`
	ArtifactVersionID                 *uuid.UUID `db:"artifact_version_id"`
	PartnerOrganizationID             *uuid.UUID `db:"partner_organization_id"`
	// Deployments are the ones an update notification listed as behind the announced version when it was sent.
	Deployments []NotificationRecordDeployment `db:"deployments"`

	DeliveryError string `db:"delivery_error"`
}

type NotificationRecordDeployment struct {
	CustomerOrganizationName *string `json:"customerOrganizationName,omitempty"`
	DeploymentTargetName     string  `json:"deploymentTargetName"`
	HelmReleaseName          *string `json:"helmReleaseName,omitempty"`
	CurrentVersionName       string  `json:"currentVersionName"`
}

type NotificationRecordWithDetails struct {
	NotificationRecord
	DeploymentTargetName           *string                  `db:"deployment_target_name"`
	CustomerOrganizationName       *string                  `db:"customer_organization_name"`
	ApplicationName                *string                  `db:"application_name"`
	ApplicationVersionName         *string                  `db:"application_version_name"`
	ArtifactName                   *string                  `db:"artifact_name"`
	ArtifactVersionName            *string                  `db:"artifact_version_name"`
	CurrentDeploymentTargetMetrics *DeploymentTargetMetrics `db:"current_deployment_target_metrics"`
}
