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
	// PreviousDeploymentRevisionID and PreviousStatusCreatedAt identify the status that triggered
	// the notification. Statuses are overwritten in place on the revision, so the pair takes the
	// place of a row id and keys the deduplication of repeated alerts.
	PreviousDeploymentRevisionID      *uuid.UUID            `db:"previous_deployment_revision_id"`
	PreviousStatusCreatedAt           *time.Time            `db:"previous_status_created_at"`
	CurrentDeploymentRevisionID       *uuid.UUID            `db:"current_deployment_revision_id"`
	CurrentStatusCreatedAt            *time.Time            `db:"current_status_created_at"`
	CurrentStatusType                 *DeploymentStatusType `db:"current_status_type"`
	CurrentStatusMessage              *string               `db:"current_status_message"`
	MetricType                        *string               `db:"metric_type"`
	DiskDevice                        *string               `db:"disk_device"`
	DiskPath                          *string               `db:"disk_path"`
	PreviousDeploymentTargetMetricsID *uuid.UUID            `db:"previous_deployment_target_metrics_id"`
	CurrentDeploymentTargetMetricsID  *uuid.UUID            `db:"current_deployment_target_metrics_id"`
	Message                           string                `db:"message" json:"message"`
}

func (r *NotificationRecord) CurrentStatus() *DeploymentRevisionStatus {
	if r.CurrentDeploymentRevisionID == nil || r.CurrentStatusCreatedAt == nil ||
		r.CurrentStatusType == nil || r.CurrentStatusMessage == nil {
		return nil
	}
	return &DeploymentRevisionStatus{
		CreatedAt:            *r.CurrentStatusCreatedAt,
		DeploymentRevisionID: *r.CurrentDeploymentRevisionID,
		Type:                 *r.CurrentStatusType,
		Message:              *r.CurrentStatusMessage,
	}
}

type NotificationRecordWithCurrentStatus struct {
	NotificationRecord
	DeploymentTargetName           *string                  `db:"deployment_target_name"`
	CustomerOrganizationName       *string                  `db:"customer_organization_name"`
	ApplicationName                *string                  `db:"application_name"`
	ApplicationVersionName         *string                  `db:"application_version_name"`
	CurrentDeploymentTargetMetrics *DeploymentTargetMetrics `db:"current_deployment_target_metrics"`
}
