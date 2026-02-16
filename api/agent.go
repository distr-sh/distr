package api

import (
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type AgentResource struct {
	Version        types.AgentVersion `json:"version"`
	Namespace      string             `json:"namespace,omitempty"`
	MetricsEnabled bool               `json:"metricsEnabled"`
	Deployments    []AgentDeployment  `json:"deployments,omitempty"`
}

type AgentRegistryAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AgentDeployment struct {
	ID           uuid.UUID                    `json:"id"`
	RevisionID   uuid.UUID                    `json:"revisionId"`
	RegistryAuth map[string]AgentRegistryAuth `json:"registryAuth"`
	LogsEnabled  bool                         `json:"logsEnabled"`
	ForceRestart bool                         `json:"forceRestart"`

	// Docker specific data

	ComposeFile []byte            `json:"composeFile"`
	EnvFile     []byte            `json:"envFile"`
	DockerType  *types.DockerType `json:"dockerType"`

	// Kubernetes specific data

	ReleaseName        string         `json:"releaseName"`
	ChartUrl           string         `json:"chartUrl"`
	ChartName          string         `json:"chartName"`
	ChartVersion       string         `json:"chartVersion"`
	Values             map[string]any `json:"values"`
	IgnoreRevisionSkew bool           `json:"ignoreRevisionSkew"`
	HelmOptions        *HelmOptions   `json:"helmOptions,omitempty"`
}

type AgentDeploymentStatus struct {
	RevisionID uuid.UUID                  `json:"revisionId"`
	Type       types.DeploymentStatusType `json:"type"`
	Message    string                     `json:"message"`
}

type AgentDeploymentTargetMetrics struct {
	CPUCoresMillis int64   `json:"cpuCoresMillis" db:"cpu_cores_millis"`
	CPUUsage       float64 `json:"cpuUsage" db:"cpu_usage"`
	MemoryBytes    int64   `json:"memoryBytes" db:"memory_bytes"`
	MemoryUsage    float64 `json:"memoryUsage" db:"memory_usage"`
}
