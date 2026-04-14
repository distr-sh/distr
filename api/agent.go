package api

import (
	"time"

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
	LogsAfter    *time.Time                   `json:"logsAfter,omitempty"`
	ForceRestart bool                         `json:"forceRestart"`

	// Docker specific data

	ComposeFile         []byte            `json:"composeFile"`
	EnvFile             []byte            `json:"envFile"`
	DockerType          *types.DockerType `json:"dockerType"`
	ImageCleanupEnabled bool              `json:"imageCleanupEnabled"`

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

type AgentDeploymentTargetMetricsRequest struct {
	CPUCoresMillis int64                        `json:"cpuCoresMillis"`
	CPUUsage       float64                      `json:"cpuUsage"`
	MemoryBytes    int64                        `json:"memoryBytes"`
	MemoryUsage    float64                      `json:"memoryUsage"`
	DiskMetrics    []DeploymentTargetDiskMetric `json:"diskMetrics,omitempty"`
}
