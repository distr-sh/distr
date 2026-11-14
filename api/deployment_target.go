package api

import (
	"fmt"
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/resource"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// EntityReference is the shape of the deprecated nested references that clients written against the
// response model send instead of a flat id.
type EntityReference struct {
	ID uuid.UUID `json:"id"`
}

func validateDeploymentType(deploymentType types.DeploymentType) error {
	if _, err := types.ParseDeploymentType(string(deploymentType)); err != nil {
		return validation.NewValidationFailedError(err.Error())
	}
	return nil
}

// ListDeploymentTargetsRequest holds the filters of the list endpoint. A deployment target matches
// when it belongs to any of the given customers, or to none if Internal is set. Omitting both
// disables the filter.
type ListDeploymentTargetsRequest struct {
	CustomerOrganizationID []string `query:"customerOrganizationId"`
	Internal               bool     `query:"internal"`
}

func (r *ListDeploymentTargetsRequest) ParseCustomerOrganizationIDs() ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for _, value := range r.CustomerOrganizationID {
		if id, err := uuid.Parse(value); err != nil {
			return nil, validation.NewValidationFailedError("customerOrganizationId must be a valid UUID")
		} else if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

type DeploymentTargetResources struct {
	CPURequest    string `json:"cpuRequest"`
	MemoryRequest string `json:"memoryRequest"`
	CPULimit      string `json:"cpuLimit"`
	MemoryLimit   string `json:"memoryLimit"`
}

type DeploymentTargetToleration struct {
	Key      string `json:"key,omitempty"`
	Operator string `json:"operator,omitempty"`
	Value    string `json:"value,omitempty"`
	Effect   string `json:"effect,omitempty"`
}

func (t DeploymentTargetToleration) validate(fldPath *field.Path) field.ErrorList {
	var errs field.ErrorList
	if t.Key != "" {
		errs = append(errs, metav1validation.ValidateLabelName(t.Key, fldPath.Child("key"))...)
	}
	switch t.Operator {
	case "", "Equal":
		if t.Key == "" {
			errs = append(errs, field.Invalid(fldPath.Child("operator"), t.Operator,
				"operator must be Exists when key is empty"))
		}
		for _, msg := range k8svalidation.IsValidLabelValue(t.Value) {
			errs = append(errs, field.Invalid(fldPath.Child("value"), t.Value, msg))
		}
	case "Exists":
		if t.Value != "" {
			errs = append(errs, field.Invalid(fldPath.Child("value"), t.Value,
				"value must be empty when operator is Exists"))
		}
	default:
		errs = append(errs, field.NotSupported(fldPath.Child("operator"), t.Operator, []string{"Equal", "Exists"}))
	}
	switch t.Effect {
	case "", "NoSchedule", "PreferNoSchedule", "NoExecute":
	default:
		errs = append(errs, field.NotSupported(fldPath.Child("effect"), t.Effect,
			[]string{"NoSchedule", "PreferNoSchedule", "NoExecute"}))
	}
	return errs
}

type DeploymentTargetSettings struct {
	Name                    string                       `json:"name"`
	MetricsEnabled          bool                         `json:"metricsEnabled"`
	ImageCleanupEnabled     bool                         `json:"imageCleanupEnabled"`
	AutomaticUpdatesEnabled bool                         `json:"automaticUpdatesEnabled"`
	DeploymentLogsEnabled   bool                         `json:"deploymentLogsEnabled"`
	DeploymentLogsAfter     *time.Time                   `json:"deploymentLogsAfter,omitempty"`
	Resources               *DeploymentTargetResources   `json:"resources,omitempty"`
	DockerEndpoint          *string                      `json:"dockerEndpoint,omitempty"`
	PodAnnotations          map[string]string            `json:"podAnnotations,omitempty"`
	NodeSelector            map[string]string            `json:"nodeSelector,omitempty"`
	Tolerations             []DeploymentTargetToleration `json:"tolerations,omitempty"`
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
		var errs field.ErrorList
		errs = append(errs, apivalidation.ValidateAnnotations(s.PodAnnotations, field.NewPath("podAnnotations"))...)
		errs = append(errs, metav1validation.ValidateLabels(s.NodeSelector, field.NewPath("nodeSelector"))...)
		for i, t := range s.Tolerations {
			errs = append(errs, t.validate(field.NewPath("tolerations").Index(i))...)
		}
		if len(errs) > 0 {
			return validation.NewValidationFailedError(errs.ToAggregate().Error())
		}
	case types.DeploymentTypeDocker:
		if s.Resources != nil {
			return validation.NewValidationFailedError("DeploymentTarget with type \"docker\" must not have resources")
		}
		if len(s.PodAnnotations) > 0 || len(s.NodeSelector) > 0 || len(s.Tolerations) > 0 {
			return validation.NewValidationFailedError(
				"DeploymentTarget with type \"docker\" must not have pod annotations, node selector or tolerations",
			)
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
	if err := validateDeploymentType(r.Type); err != nil {
		return err
	}
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
	}
	return r.DeploymentTargetSettings.Validate(r.Type)
}

type UpdateDeploymentTargetRequest struct {
	DeploymentTargetSettings
	ControllerVersionID *uuid.UUID `json:"controllerVersionId,omitempty"`
	// Deprecated: ControllerVersion is what clients sent before ControllerVersionID existed.
	ControllerVersion *EntityReference `json:"controllerVersion,omitempty" deprecated:"true"`
}

type ControllerVersion struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`
}

type DeploymentTargetResponse struct {
	ID                          uuid.UUID                    `json:"id"`
	CreatedAt                   time.Time                    `json:"createdAt"`
	Name                        string                       `json:"name"`
	Type                        types.DeploymentType         `json:"type"`
	ReconnectPending            bool                         `json:"reconnectPending"`
	Namespace                   *string                      `json:"namespace,omitempty"`
	Scope                       *types.DeploymentTargetScope `json:"scope,omitempty"`
	CustomerOrganizationID      *uuid.UUID                   `json:"customerOrganizationId,omitempty"`
	ReportedControllerVersionID *uuid.UUID                   `json:"reportedControllerVersionId,omitempty"`
	MetricsEnabled              bool                         `json:"metricsEnabled"`
	ImageCleanupEnabled         bool                         `json:"imageCleanupEnabled"`
	AutohealEnabled             bool                         `json:"autohealEnabled"`
	AutomaticUpdatesEnabled     bool                         `json:"automaticUpdatesEnabled"`
	DeploymentLogsEnabled       bool                         `json:"deploymentLogsEnabled"`
	DeploymentLogsAfter         *time.Time                   `json:"deploymentLogsAfter,omitempty"`
	Resources                   *DeploymentTargetResources   `json:"resources,omitempty"`
	DockerEndpoint              *string                      `json:"dockerEndpoint,omitempty"`
	PodAnnotations              map[string]string            `json:"podAnnotations,omitempty"`
	NodeSelector                map[string]string            `json:"nodeSelector,omitempty"`
	Tolerations                 []DeploymentTargetToleration `json:"tolerations,omitempty"`
	CustomerOrganization        *CustomerOrganization        `json:"customerOrganization,omitempty"`
	Deployments                 []DeploymentResponse         `json:"deployments"`
	ControllerVersion           ControllerVersion            `json:"controllerVersion"`
}

type DeploymentResponse struct {
	ID                                 uuid.UUID                  `json:"id"`
	CreatedAt                          time.Time                  `json:"createdAt"`
	DeploymentTargetID                 uuid.UUID                  `json:"deploymentTargetId"`
	ReleaseName                        *string                    `json:"releaseName,omitempty"`
	ApplicationEntitlementID           *uuid.UUID                 `json:"applicationEntitlementId,omitempty"`
	DockerType                         *types.DockerType          `json:"dockerType,omitempty"`
	AutomaticApplicationUpdatesEnabled bool                       `json:"automaticApplicationUpdatesEnabled"`
	DeploymentRevisionID               uuid.UUID                  `json:"deploymentRevisionId"`
	DeploymentRevisionCreatedAt        time.Time                  `json:"deploymentRevisionCreatedAt"`
	DeploymentRevisionCreatedBy        *DeploymentRevisionCreator `json:"deploymentRevisionCreatedBy,omitempty"`
	// Deprecated: Use [DeploymentResponse.Application] instead
	ApplicationID uuid.UUID `json:"applicationId" deprecated:"true"`
	// Deprecated: Use [DeploymentResponse.Application] instead
	ApplicationName               string                    `json:"applicationName" deprecated:"true"`
	Application                   ApplicationResponse       `json:"application"`
	ApplicationVersionID          uuid.UUID                 `json:"applicationVersionId"`
	ApplicationVersionName        string                    `json:"applicationVersionName"`
	ApplicationLink               string                    `json:"applicationLink"`
	ValuesYaml                    []byte                    `json:"valuesYaml,omitempty"`
	EnvFileData                   []byte                    `json:"envFileData,omitempty"`
	LatestStatus                  *DeploymentRevisionStatus `json:"latestStatus,omitempty"`
	ForceRestart                  bool                      `json:"forceRestart"`
	IgnoreRevisionSkew            bool                      `json:"ignoreRevisionSkew"`
	HelmOptions                   *HelmOptions              `json:"helmOptions,omitempty"`
	CurrentDeploymentRevisionID   *uuid.UUID                `json:"currentDeploymentRevisionId,omitempty"`
	CurrentStatus                 *DeploymentRevisionStatus `json:"currentStatus,omitempty"`
	CurrentApplicationVersionID   *uuid.UUID                `json:"currentApplicationVersionId,omitempty"`
	CurrentApplicationVersionName *string                   `json:"currentApplicationVersionName,omitempty"`
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
	ReleaseName              *string                         `json:"releaseName" description:"Ignored when updating a deployment, cannot change later"` //nolint:lll
	ValuesYaml               []byte                          `json:"valuesYaml"`
	DockerType               *types.DockerType               `json:"dockerType" description:"Defaults to compose on docker targets, cannot change later"` //nolint:lll
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
