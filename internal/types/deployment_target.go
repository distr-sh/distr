package types

import (
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/resource"
)

type DeploymentTarget struct {
	ID                     uuid.UUID                  `db:"id" json:"id"`
	CreatedAt              time.Time                  `db:"created_at" json:"createdAt"`
	Name                   string                     `db:"name" json:"name"`
	Type                   DeploymentType             `db:"type" json:"type"`
	AccessKeySalt          *[]byte                    `db:"access_key_salt" json:"-"`
	AccessKeyHash          *[]byte                    `db:"access_key_hash" json:"-"`
	Namespace              *string                    `db:"namespace" json:"namespace,omitempty"`
	Scope                  *DeploymentTargetScope     `db:"scope" json:"scope,omitempty"`
	OrganizationID         uuid.UUID                  `db:"organization_id" json:"-"`
	CustomerOrganizationID *uuid.UUID                 `db:"customer_organization_id" json:"customerOrganizationId,omitempty"` //nolint:lll
	AgentVersionID         *uuid.UUID                 `db:"agent_version_id" json:"-"`
	ReportedAgentVersionID *uuid.UUID                 `db:"reported_agent_version_id" json:"reportedAgentVersionId,omitempty"` //nolint:lll
	MetricsEnabled         bool                       `db:"metrics_enabled" json:"metricsEnabled"`
	Resources              *DeploymentTargetResources `db:"resources" json:"resources,omitempty"`
}

type DeploymentTargetResources struct {
	CPURequest    string `json:"cpuRequest"`
	MemoryRequest string `json:"memoryRequest"`
	CPULimit      string `json:"cpuLimit"`
	MemoryLimit   string `json:"memoryLimit"`
}

func (dt *DeploymentTarget) Validate() error {
	switch dt.Type {
	case DeploymentTypeKubernetes:
		if dt.Namespace == nil || *dt.Namespace == "" {
			return validation.NewValidationFailedError(
				"DeploymentTarget with type \"kubernetes\" must not have empty namespace",
			)
		}
		if dt.Scope == nil {
			return validation.NewValidationFailedError("DeploymentTarget with type \"kubernetes\" must not have empty scope")
		}
		if dt.Resources != nil {
			if _, err := resource.ParseQuantity(dt.Resources.CPULimit); err != nil {
				return validation.NewValidationFailedError(fmt.Sprintf("failed to parse CPU limit: %s", err))
			}
			if _, err := resource.ParseQuantity(dt.Resources.MemoryLimit); err != nil {
				return validation.NewValidationFailedError(fmt.Sprintf("failed to parse memory limit: %s", err))
			}
			if _, err := resource.ParseQuantity(dt.Resources.CPURequest); err != nil {
				return validation.NewValidationFailedError(fmt.Sprintf("failed to parse CPU request: %s", err))
			}
			if _, err := resource.ParseQuantity(dt.Resources.MemoryRequest); err != nil {
				return validation.NewValidationFailedError(fmt.Sprintf("failed to parse memory request: %s", err))
			}
		}
	case DeploymentTypeDocker:
		if dt.Resources != nil {
			return validation.NewValidationFailedError("DeploymentTarget with type \"docker\" must not have resources")
		}
	default:
		return validation.NewValidationFailedError("invalid deployment target type")
	}
	return nil
}

type DeploymentTargetFull struct {
	DeploymentTarget
	CustomerOrganization *CustomerOrganization          `db:"customer_organization" json:"customerOrganization,omitempty"`
	CurrentStatus        *DeploymentTargetStatus        `db:"current_status" json:"currentStatus,omitempty"`
	Deployments          []DeploymentWithLatestRevision `db:"-" json:"deployments"`
	AgentVersion         AgentVersion                   `db:"agent_version" json:"agentVersion"`
}
