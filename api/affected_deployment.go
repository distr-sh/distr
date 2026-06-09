package api

import "github.com/google/uuid"

type AffectedDeployment struct {
	DeploymentTargetID   uuid.UUID `json:"deploymentTargetId"`
	DeploymentTargetName string    `json:"deploymentTargetName"`
	DeploymentID         uuid.UUID `json:"deploymentId"`
	ApplicationName      string    `json:"applicationName"`
}

type AffectedDeploymentsConflictResponse struct {
	AffectedDeployments []AffectedDeployment `json:"affectedDeployments"`
}
