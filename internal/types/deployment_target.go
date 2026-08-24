package types

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode"

	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/resource"
)

type DeploymentTarget struct {
	ID                      uuid.UUID                  `db:"id" json:"id"`
	CreatedAt               time.Time                  `db:"created_at" json:"createdAt"`
	Name                    string                     `db:"name" json:"name"`
	Type                    DeploymentType             `db:"type" json:"type"`
	AccessKeySalt           *[]byte                    `db:"access_key_salt" json:"-"`
	AccessKeyHash           *[]byte                    `db:"access_key_hash" json:"-"`
	Namespace               *string                    `db:"namespace" json:"namespace,omitempty"`
	Scope                   *DeploymentTargetScope     `db:"scope" json:"scope,omitempty"`
	OrganizationID          uuid.UUID                  `db:"organization_id" json:"-"`
	CustomerOrganizationID  *uuid.UUID                 `db:"customer_organization_id" json:"customerOrganizationId,omitempty"` //nolint:lll
	AgentVersionID          *uuid.UUID                 `db:"agent_version_id" json:"-"`
	ReportedAgentVersionID  *uuid.UUID                 `db:"reported_agent_version_id" json:"reportedAgentVersionId,omitempty"` //nolint:lll
	MetricsEnabled          bool                       `db:"metrics_enabled" json:"metricsEnabled"`
	ImageCleanupEnabled     bool                       `db:"image_cleanup_enabled" json:"imageCleanupEnabled"`
	AutohealEnabled         bool                       `db:"autoheal_enabled" json:"autohealEnabled"`
	AutomaticUpdatesEnabled bool                       `db:"automatic_updates_enabled" json:"automaticUpdatesEnabled"`
	DeploymentLogsEnabled   bool                       `db:"deployment_logs_enabled" json:"deploymentLogsEnabled"`
	DeploymentLogsAfter     *time.Time                 `db:"deployment_logs_after" json:"deploymentLogsAfter,omitempty"`
	Resources               *DeploymentTargetResources `db:"resources" json:"resources,omitempty"`
	DockerEndpoint          *string                    `db:"docker_endpoint" json:"dockerEndpoint,omitempty"`
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
		if dt.AutohealEnabled {
			return validation.NewValidationFailedError(
				"DeploymentTarget with type \"kubernetes\" must not have autoheal enabled",
			)
		}
		if dt.Namespace == nil || *dt.Namespace == "" {
			return validation.NewValidationFailedError(
				"DeploymentTarget with type \"kubernetes\" must not have empty namespace",
			)
		}
		if dt.Scope == nil {
			return validation.NewValidationFailedError("DeploymentTarget with type \"kubernetes\" must not have empty scope")
		}
		if dt.ImageCleanupEnabled {
			return validation.NewValidationFailedError(
				"image cleanup is not supported on DeploymentTarget with type \"kubernetes\"")
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
	return ValidateDockerEndpoint(dt.DockerEndpoint, dt.Type)
}

const DefaultDockerSocketPath = "/var/run/docker.sock"

// ParseDockerEndpoint only supports unix sockets because the endpoint is applied by bind-mounting
// the socket into the agent container.
func ParseDockerEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid docker endpoint: %w", err)
	}
	if u.Scheme != "unix" {
		return "", errors.New("docker endpoint must have scheme \"unix\"")
	}
	if u.Host != "" {
		return "", errors.New("docker endpoint must not have a host")
	}
	if !path.IsAbs(u.Path) {
		return "", errors.New("docker endpoint must have an absolute socket path")
	}
	// The path is rendered into Compose's short "SOURCE:TARGET" volume syntax.
	if strings.Contains(u.Path, ":") || strings.ContainsFunc(u.Path, unicode.IsSpace) {
		return "", errors.New("docker endpoint socket path must not contain a colon or whitespace")
	}
	return u.Path, nil
}

func (dt *DeploymentTarget) DockerSocketPath() string {
	if dt.DockerEndpoint != nil {
		if socketPath, err := ParseDockerEndpoint(*dt.DockerEndpoint); err == nil {
			return socketPath
		}
	}
	return DefaultDockerSocketPath
}

// ValidateDockerEndpoint takes the type separately because updates do not carry it in the body.
func ValidateDockerEndpoint(endpoint *string, deploymentType DeploymentType) error {
	if endpoint == nil {
		return nil
	}
	if deploymentType != DeploymentTypeDocker {
		return validation.NewValidationFailedError(
			fmt.Sprintf("DeploymentTarget with type %q must not have a docker endpoint", deploymentType),
		)
	}
	if _, err := ParseDockerEndpoint(*endpoint); err != nil {
		return validation.NewValidationFailedError(err.Error())
	}
	return nil
}

type DeploymentTargetFull struct {
	DeploymentTarget
	CustomerOrganization *CustomerOrganization          `db:"customer_organization" json:"customerOrganization,omitempty"`
	Deployments          []DeploymentWithLatestRevision `db:"-" json:"deployments"`
	AgentVersion         AgentVersion                   `db:"agent_version" json:"agentVersion"`
}
