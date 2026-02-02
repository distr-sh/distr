package api

import (
	"time"

	"github.com/google/uuid"
)

type NotificationRecord struct {
	ID                                          uuid.UUID  `json:"id"`
	CreatedAt                                   time.Time  `json:"createdAt"`
	DeploymentTargetID                          *uuid.UUID `json:"deploymentTargetId"`
	DeploymentStatusNotificationConfigurationID *uuid.UUID `json:"deploymentStatusNotificationConfigurationId,omitempty"`
	PreviousDeploymentRevisionStatusID          *uuid.UUID `json:"previousDeploymentStatusId,omitempty"`
	CurrentDeploymentRevisionStatusID           *uuid.UUID `json:"currentDeploymentStatusId,omitempty"`
	Message                                     string     `json:"message"`
}

type NotificationRecordWithCurrentStatus struct {
	NotificationRecord
	DeploymentTargetName            *string                   `json:"deploymentTargetName,omitempty"`
	CurrentDeploymentRevisionStatus *DeploymentRevisionStatus `json:"currentDeploymentRevisionStatus,omitempty"`
}
