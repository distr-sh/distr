package api

import (
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type DeploymentTargetAccessTokenResponse struct {
	ConnectURL     string    `json:"connectUrl"`
	TargetID       uuid.UUID `json:"targetId"`
	TargetSecret   string    `json:"targetSecret"`
	ConnectCommand string    `json:"connectCommand"`
}

type DeploymentRequest struct {
	DeploymentID             *uuid.UUID        `json:"deploymentId"`
	DeploymentTargetID       uuid.UUID         `json:"deploymentTargetId"`
	ApplicationVersionID     uuid.UUID         `json:"applicationVersionId"`
	ApplicationEntitlementID *uuid.UUID        `json:"applicationEntitlementId"`
	ReleaseName              *string           `json:"releaseName"`
	ValuesYaml               []byte            `json:"valuesYaml"`
	DockerType               *types.DockerType `json:"dockerType"`
	EnvFileData              []byte            `json:"envFileData"`
	ForceRestart             bool              `json:"forceRestart"`
	IgnoreRevisionSkew       bool              `json:"ignoreRevisionSkew"`
	HelmOptions              *HelmOptions      `json:"helmOptions,omitempty"`
	TofuVars                 map[string]any    `json:"tofuVars,omitempty"`
	TofuBackendConfig        map[string]string `json:"tofuBackendConfig,omitempty"`
	TofuVersion              *string           `json:"tofuVersion,omitempty"`
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
}
