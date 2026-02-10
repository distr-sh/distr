package types

import (
	"time"

	"github.com/google/uuid"
)

type NotificationRecord struct {
	ID                                 uuid.UUID  `db:"id"`
	CreatedAt                          time.Time  `db:"created_at"`
	OrganizationID                     uuid.UUID  `db:"organization_id"`
	CustomerOrganizationID             *uuid.UUID `db:"customer_organization_id"`
	DeploymentTargetID                 *uuid.UUID `db:"deployment_target_id"`
	AlertConfigurationID               *uuid.UUID `db:"alert_configuration_id"`
	PreviousDeploymentRevisionStatusID *uuid.UUID `db:"previous_deployment_revision_status_id"`
	CurrentDeploymentRevisionStatusID  *uuid.UUID `db:"current_deployment_revision_status_id"`
	Message                            string     `db:"message" json:"message"`
}

type NotificationRecordWithCurrentStatus struct {
	NotificationRecord
	DeploymentTargetName            *string                   `db:"deployment_target_name"`
	CustomerOrganizationName        *string                   `db:"customer_organization_name"`
	ApplicationName                 *string                   `db:"application_name"`
	ApplicationVersionName          *string                   `db:"application_version_name"`
	CurrentDeploymentRevisionStatus *DeploymentRevisionStatus `db:"current_deployment_revision_status"`
}
