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
	NotificationRecordTypeNewVersion      NotificationRecordType = "new_version"
)

type NotificationSourceType string

const (
	NotificationSourceTypeAlert       NotificationSourceType = "alert"
	NotificationSourceTypeApplication NotificationSourceType = "application"
	NotificationSourceTypeArtifact    NotificationSourceType = "artifact"
)

type NotificationRecord struct {
	ID                     uuid.UUID              `db:"id"`
	CreatedAt              time.Time              `db:"created_at"`
	OrganizationID         uuid.UUID              `db:"organization_id"`
	CustomerOrganizationID *uuid.UUID             `db:"customer_organization_id"`
	UserAccountID          *uuid.UUID             `db:"user_account_id"`
	SourceType             NotificationSourceType `db:"source_type"`
	SourceConfigurationID  *uuid.UUID             `db:"source_configuration_id"`
	SubjectID              *uuid.UUID             `db:"subject_id"`
	Type                   NotificationRecordType `db:"type"`
	// DeploymentRevisionID is the revision a deployment status alert is about. It is a column rather
	// than part of Details because resolving stale warnings joins on it.
	DeploymentRevisionID *uuid.UUID `db:"deployment_revision_id"`
	// ResolvedAt is set on a stale warning once the agent reports again. An unresolved warning
	// suppresses further warnings for the same deployment and alert configuration.
	ResolvedAt    *time.Time                `db:"resolved_at"`
	Details       NotificationRecordDetails `db:"details"`
	DeliveryError string                    `db:"delivery_error"`
}

// NotificationRecordDetails is the trigger-specific payload of a record, stored in a single JSONB
// column so that a new notification trigger does not add a column to the shared history table. It
// carries json tags rather than db tags for that reason, and it holds the names of everything it
// refers to, because a record has to stay displayable after its subject has been deleted.
type NotificationRecordDetails struct {
	Summary                  string  `json:"summary,omitempty"`
	CustomerOrganizationName *string `json:"customerOrganizationName,omitempty"`

	DeploymentTargetName   *string         `json:"deploymentTargetName,omitempty"`
	ApplicationName        *string         `json:"applicationName,omitempty"`
	ApplicationType        *DeploymentType `json:"applicationType,omitempty"`
	ApplicationVersionName *string         `json:"applicationVersionName,omitempty"`
	ArtifactName           *string         `json:"artifactName,omitempty"`
	ArtifactVersionName    *string         `json:"artifactVersionName,omitempty"`

	MetricType *string `json:"metricType,omitempty"`
	DiskDevice *string `json:"diskDevice,omitempty"`
	DiskPath   *string `json:"diskPath,omitempty"`

	PreviousDeploymentTargetMetricsID *uuid.UUID `json:"previousDeploymentTargetMetricsId,omitempty"`
	CurrentDeploymentTargetMetricsID  *uuid.UUID `json:"currentDeploymentTargetMetricsId,omitempty"`

	// Deployments are the ones a notification announced an update for, which is more than one
	// whenever a recipient sees several affected deployments.
	Deployments []NotificationRecordDeployment `json:"deployments,omitempty"`
}

type NotificationRecordDeployment struct {
	CustomerOrganizationName *string `json:"customerOrganizationName,omitempty"`
	DeploymentTargetName     string  `json:"deploymentTargetName"`
	DeploymentName           string  `json:"deploymentName"`
	CurrentVersionName       *string `json:"currentVersionName,omitempty"`
}
