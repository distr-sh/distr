package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type NotificationRecord struct {
	ID                    uuid.UUID                    `json:"id"`
	CreatedAt             time.Time                    `json:"createdAt"`
	UserAccountID         *uuid.UUID                   `json:"userAccountId,omitempty"`
	SourceType            types.NotificationSourceType `json:"sourceType"`
	SourceConfigurationID *uuid.UUID                   `json:"sourceConfigurationId,omitempty"`
	SubjectID             *uuid.UUID                   `json:"subjectId,omitempty"`
	Type                  types.NotificationRecordType `json:"type"`
	Details               NotificationRecordDetails    `json:"details"`
	DeliveryError         string                       `json:"deliveryError"`
}

type NotificationRecordDetails struct {
	Summary                  string  `json:"summary,omitempty"`
	CustomerOrganizationName *string `json:"customerOrganizationName,omitempty"`

	DeploymentTargetName   *string               `json:"deploymentTargetName,omitempty"`
	ApplicationName        *string               `json:"applicationName,omitempty"`
	ApplicationType        *types.DeploymentType `json:"applicationType,omitempty"`
	ApplicationVersionName *string               `json:"applicationVersionName,omitempty"`
	ArtifactName           *string               `json:"artifactName,omitempty"`
	ArtifactVersionName    *string               `json:"artifactVersionName,omitempty"`

	Deployments []NotificationRecordDeployment `json:"deployments,omitempty"`
}

type NotificationRecordDeployment struct {
	CustomerOrganizationName *string `json:"customerOrganizationName,omitempty"`
	DeploymentTargetName     string  `json:"deploymentTargetName"`
	DeploymentName           string  `json:"deploymentName"`
	CurrentVersionName       *string `json:"currentVersionName,omitempty"`
}
