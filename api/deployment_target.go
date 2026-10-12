package api

import (
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/resource"
)

// EntityReference is the shape of the deprecated nested references that clients written against the
// response model send instead of a flat id.
type EntityReference struct {
	ID uuid.UUID `json:"id"`
}

type DeploymentTargetResources struct {
	CPURequest    string `json:"cpuRequest"`
	MemoryRequest string `json:"memoryRequest"`
	CPULimit      string `json:"cpuLimit"`
	MemoryLimit   string `json:"memoryLimit"`
}

type DeploymentTargetSettings struct {
	Name                    string                     `json:"name"`
	MetricsEnabled          bool                       `json:"metricsEnabled"`
	ImageCleanupEnabled     bool                       `json:"imageCleanupEnabled"`
	AutomaticUpdatesEnabled bool                       `json:"automaticUpdatesEnabled"`
	DeploymentLogsEnabled   bool                       `json:"deploymentLogsEnabled"`
	DeploymentLogsAfter     *time.Time                 `json:"deploymentLogsAfter,omitempty"`
	Resources               *DeploymentTargetResources `json:"resources,omitempty"`
	DockerEndpoint          *string                    `json:"dockerEndpoint,omitempty"`
}

// Validate takes the type as an argument because an update never changes it, so an update request
// does not carry the type to validate against.
func (s *DeploymentTargetSettings) Validate(deploymentType types.DeploymentType) error {
	switch deploymentType {
	case types.DeploymentTypeKubernetes:
		if s.ImageCleanupEnabled {
			return validation.NewValidationFailedError(
				"image cleanup is not supported on DeploymentTarget with type \"kubernetes\"")
		}
		if s.Resources != nil {
			if _, err := resource.ParseQuantity(s.Resources.CPULimit); err != nil {
				return validation.NewValidationFailedError(fmt.Sprintf("failed to parse CPU limit: %s", err))
			}
			if _, err := resource.ParseQuantity(s.Resources.MemoryLimit); err != nil {
				return validation.NewValidationFailedError(fmt.Sprintf("failed to parse memory limit: %s", err))
			}
			if _, err := resource.ParseQuantity(s.Resources.CPURequest); err != nil {
				return validation.NewValidationFailedError(fmt.Sprintf("failed to parse CPU request: %s", err))
			}
			if _, err := resource.ParseQuantity(s.Resources.MemoryRequest); err != nil {
				return validation.NewValidationFailedError(fmt.Sprintf("failed to parse memory request: %s", err))
			}
		}
	case types.DeploymentTypeDocker:
		if s.Resources != nil {
			return validation.NewValidationFailedError("DeploymentTarget with type \"docker\" must not have resources")
		}
	}
	if s.DockerEndpoint != nil {
		if deploymentType != types.DeploymentTypeDocker {
			return validation.NewValidationFailedError(
				fmt.Sprintf("DeploymentTarget with type %q must not have a docker endpoint", deploymentType),
			)
		}
		if _, err := types.ParseDockerEndpoint(*s.DockerEndpoint); err != nil {
			return validation.NewValidationFailedError(err.Error())
		}
	}
	return nil
}

type CreateDeploymentTargetRequest struct {
	DeploymentTargetSettings
	Type                   types.DeploymentType         `json:"type"`
	Namespace              *string                      `json:"namespace,omitempty"`
	Scope                  *types.DeploymentTargetScope `json:"scope,omitempty"`
	AutohealEnabled        bool                         `json:"autohealEnabled"`
	CustomerOrganizationID *uuid.UUID                   `json:"customerOrganizationId,omitempty"`
	// Deprecated: CustomerOrganization is what clients sent before CustomerOrganizationID existed.
	CustomerOrganization *EntityReference `json:"customerOrganization,omitempty" deprecated:"true"`
}

func (r *CreateDeploymentTargetRequest) Validate() error {
	switch r.Type {
	case types.DeploymentTypeKubernetes:
		if r.AutohealEnabled {
			return validation.NewValidationFailedError(
				"DeploymentTarget with type \"kubernetes\" must not have autoheal enabled",
			)
		}
		if r.Namespace == nil || *r.Namespace == "" {
			return validation.NewValidationFailedError(
				"DeploymentTarget with type \"kubernetes\" must not have empty namespace",
			)
		}
		if r.Scope == nil {
			return validation.NewValidationFailedError("DeploymentTarget with type \"kubernetes\" must not have empty scope")
		}
	case types.DeploymentTypeDocker:
		if r.Namespace != nil {
			return validation.NewValidationFailedError("DeploymentTarget with type \"docker\" must not have a namespace")
		}
		if r.Scope != nil {
			return validation.NewValidationFailedError("DeploymentTarget with type \"docker\" must not have a scope")
		}
	default:
		return validation.NewValidationFailedError("invalid deployment target type")
	}
	return r.DeploymentTargetSettings.Validate(r.Type)
}

type UpdateDeploymentTargetRequest struct {
	DeploymentTargetSettings
	ControllerVersionID *uuid.UUID `json:"controllerVersionId,omitempty"`
	// Deprecated: ControllerVersion is what clients sent before ControllerVersionID existed.
	ControllerVersion *EntityReference `json:"controllerVersion,omitempty" deprecated:"true"`
}

type DeploymentTargetAccessTokenResponse struct {
	ConnectURL     string    `json:"connectUrl"`
	TargetID       uuid.UUID `json:"targetId"`
	TargetSecret   string    `json:"targetSecret"`
	ConnectCommand string    `json:"connectCommand"`
}

type DeploymentRequest struct {
	DeploymentID             *uuid.UUID                      `json:"deploymentId"`
	DeploymentTargetID       uuid.UUID                       `json:"deploymentTargetId"`
	ApplicationVersionID     uuid.UUID                       `json:"applicationVersionId"`
	ApplicationEntitlementID *uuid.UUID                      `json:"applicationEntitlementId"`
	ReleaseName              *string                         `json:"releaseName"`
	ValuesYaml               []byte                          `json:"valuesYaml"`
	DockerType               *types.DockerType               `json:"dockerType"`
	EnvFileData              []byte                          `json:"envFileData"`
	ForceRestart             bool                            `json:"forceRestart"`
	IgnoreRevisionSkew       bool                            `json:"ignoreRevisionSkew"`
	HelmOptions              *HelmOptions                    `json:"helmOptions,omitempty"`
	ValuesHash               []byte                          `json:"-"`
	CreatedByUserAccountID   *uuid.UUID                      `json:"-"`
	Trigger                  types.DeploymentRevisionTrigger `json:"-"`

	// AutomaticApplicationUpdatesEnabled leaves an existing deployment's setting alone when it is
	// absent, so that a client written before the field existed cannot turn automatic updates off.
	AutomaticApplicationUpdatesEnabled *bool `json:"automaticApplicationUpdatesEnabled,omitempty"`
}

type PatchDeploymentRequest struct {
	AutomaticApplicationUpdatesEnabled *bool `json:"automaticApplicationUpdatesEnabled,omitempty"`
}

func (d *DeploymentRequest) GetValuesYAML() []byte {
	return d.ValuesYaml
}

func (d *DeploymentRequest) GetEnvFileData() []byte {
	return d.EnvFileData
}

type HelmOptions struct {
	Timeout           types.Duration `json:"timeout"`
	WaitStrategy      string         `json:"waitStrategy"`
	RollbackOnFailure bool           `json:"rollbackOnFailure"`
	CleanupOnFailure  bool           `json:"cleanupOnFailure"`
	ForceConflicts    bool           `json:"forceConflicts"`
}
