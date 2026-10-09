package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type ControllerResource struct {
	Version               types.ControllerVersion `json:"version"`
	Namespace             string                  `json:"namespace,omitempty"`
	MetricsEnabled        bool                    `json:"metricsEnabled"`
	DeploymentLogsEnabled bool                    `json:"deploymentLogsEnabled"`
	DeploymentLogsAfter   *time.Time              `json:"deploymentLogsAfter,omitempty"`
	Deployments           []ControllerDeployment  `json:"deployments,omitempty"`
}

type ControllerRegistryAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ControllerDeployment struct {
	ID           uuid.UUID                         `json:"id"`
	RevisionID   uuid.UUID                         `json:"revisionId"`
	RegistryAuth map[string]ControllerRegistryAuth `json:"registryAuth"`

	// Deprecated: Use DeploymentLogsEnabled in [ControllerResource]
	LogsEnabled bool `json:"logsEnabled"`

	ForceRestart bool `json:"forceRestart"`

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

type ControllerDeploymentStatus struct {
	RevisionID uuid.UUID                  `json:"revisionId"`
	Type       types.DeploymentStatusType `json:"type"`
	Message    string                     `json:"message"`
}

type ControllerDeploymentTargetMetricsRequest struct {
	CPUCoresMillis           int64   `json:"cpuCoresMillis"`
	CPUUsage                 float64 `json:"cpuUsage"`
	MemoryBytes              int64   `json:"memoryBytes"`
	MemoryUsage              float64 `json:"memoryUsage"`
	ControllerCPUUsageMillis *int64  `json:"controllerCpuUsageMillis,omitempty"`
	ControllerMemoryBytes    *int64  `json:"controllerMemoryBytes,omitempty"`
	ControllerLogBytes       *int64  `json:"controllerLogBytes,omitempty"`
	// Deprecated: AgentCPUUsageMillis is what controllers released before the rename send instead of
	// ControllerCPUUsageMillis.
	AgentCPUUsageMillis *int64 `json:"agentCpuUsageMillis,omitempty" deprecated:"true"`
	// Deprecated: AgentMemoryBytes is what controllers released before the rename send instead of
	// ControllerMemoryBytes.
	AgentMemoryBytes *int64 `json:"agentMemoryBytes,omitempty" deprecated:"true"`
	// Deprecated: AgentLogBytes is what controllers released before the rename send instead of
	// ControllerLogBytes.
	AgentLogBytes *int64                       `json:"agentLogBytes,omitempty" deprecated:"true"`
	ImageBytes    *int64                       `json:"imageBytes,omitempty"`
	DiskMetrics   []DeploymentTargetDiskMetric `json:"diskMetrics,omitempty"`
}
