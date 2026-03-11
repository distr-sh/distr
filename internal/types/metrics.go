package types

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentTargetStatusMetricsItem struct {
	OrganizationName                string
	CustomerOrganizationName        *string
	DeploymentTargetName            string
	DeploymentTargetStatusTimestamp *time.Time
}

type DeploymentStatusMetricsItem struct {
	OrganizationName          string
	CustomerOrganizationName  *string
	DeploymentTargetName      string
	DeploymentID              uuid.UUID
	ApplicationName           string
	ApplicationVersionName    string
	DeploymentStatusTimestamp *time.Time
	DeploymentStatusType      *DeploymentStatusType
}
