package types

import (
	"time"

	"github.com/google/uuid"
)

type NotificationRecord struct {
	ID                                          uuid.UUID  `db:"id" json:"id"`
	CreatedAt                                   time.Time  `db:"created_at" json:"createdAt"`
	OrganizationID                              uuid.UUID  `db:"organization_id" json:"-"`
	CustomerOrganizationID                      *uuid.UUID `db:"customer_organization_id" json:"-"`
	DeploymentTargetID                          *uuid.UUID `db:"deployment_target_id" json:"deploymentTargetId"`
	DeploymentStatusNotificationConfigurationID *uuid.UUID `db:"deployment_status_notification_configuration_id" json:"deploymentStatusNotificationConfigurationId,omitempty"` //nolint:lll
	PreviousDeploymentRevisionStatusID          *uuid.UUID `db:"previous_deployment_revision_status_id" json:"previousDeploymentStatusId,omitempty"`                           //nolint:lll
	CurrentDeploymentRevisionStatusID           *uuid.UUID `db:"current_deployment_revision_status_id" json:"currentDeploymentStatusId,omitempty"`                             //nolint:lll
	Message                                     string     `db:"message" json:"message"`
}

type NotificationRecordWithCurrentStatus struct {
	NotificationRecord
	CurrentDeploymentRevisionStatus *DeploymentRevisionStatus `db:"current_deployment_revision_status" json:"currentDeploymentRevisionStatus,omitempty"` //nolint:lll
}
