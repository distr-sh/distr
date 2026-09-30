package api

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentTargetMetrics struct {
	DeploymentTargetID       uuid.UUID `json:"deploymentTargetId"`
	CreatedAt                time.Time `json:"createdAt"`
	CPUCoresMillis           int64     `json:"cpuCoresMillis"`
	CPUUsage                 float64   `json:"cpuUsage"`
	MemoryBytes              int64     `json:"memoryBytes"`
	MemoryUsage              float64   `json:"memoryUsage"`
	ControllerCPUUsageMillis *int64    `json:"controllerCpuUsageMillis,omitempty"`
	ControllerMemoryBytes    *int64    `json:"controllerMemoryBytes,omitempty"`
	ControllerLogBytes       *int64    `json:"controllerLogBytes,omitempty"`
	// Deprecated: AgentCPUUsageMillis repeats ControllerCPUUsageMillis under its former JSON name.
	AgentCPUUsageMillis *int64 `json:"agentCpuUsageMillis,omitempty"`
	// Deprecated: AgentMemoryBytes repeats ControllerMemoryBytes under its former JSON name.
	AgentMemoryBytes *int64 `json:"agentMemoryBytes,omitempty"`
	// Deprecated: AgentLogBytes repeats ControllerLogBytes under its former JSON name.
	AgentLogBytes *int64 `json:"agentLogBytes,omitempty"`
	// ImageBytes is the size of the docker image store, with layers shared between images counted
	// once. It is always nil in kubernetes.
	ImageBytes  *int64                       `json:"imageBytes,omitempty"`
	DiskMetrics []DeploymentTargetDiskMetric `json:"diskMetrics,omitempty"`
}

type ControllerDeploymentResourceMetricsRequest struct {
	Resources []DeploymentResourceMetric `json:"resources"`
}

type DeploymentResourceMetrics struct {
	DeploymentID uuid.UUID                  `json:"deploymentId"`
	CreatedAt    time.Time                  `json:"createdAt"`
	Resources    []DeploymentResourceMetric `json:"resources"`
}

type DeploymentResourceMetric struct {
	// Resource is the compose service name or the kubernetes workload (e.g. "Deployment/foo").
	Resource string `json:"resource"`
	// Container is the container name (docker) or "podName/containerName" (kubernetes).
	Container      string `json:"container"`
	CPUUsageMillis int64  `json:"cpuUsageMillis"`
	MemoryBytes    int64  `json:"memoryBytes"`
	// CPULimitMillis and MemoryLimitBytes are nil when the container has no limit configured.
	CPULimitMillis   *int64 `json:"cpuLimitMillis,omitempty"`
	MemoryLimitBytes *int64 `json:"memoryLimitBytes,omitempty"`
	// LogBytes is the size of the container's log files on disk. It is nil when the container
	// uses a log driver that does not write files (e.g. journald), and always nil in kubernetes.
	LogBytes *int64 `json:"logBytes,omitempty"`
}

type DeploymentTargetDiskMetric struct {
	Device     string `json:"device"`
	Path       string `json:"path"`
	FsType     string `json:"fsType"`
	BytesTotal int64  `json:"bytesTotal"`
	BytesUsed  int64  `json:"bytesUsed"`
}
