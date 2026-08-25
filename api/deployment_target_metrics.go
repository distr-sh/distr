package api

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentTargetMetrics struct {
	DeploymentTargetID  uuid.UUID                    `json:"deploymentTargetId"`
	CreatedAt           time.Time                    `json:"createdAt"`
	CPUCoresMillis      int64                        `json:"cpuCoresMillis"`
	CPUUsage            float64                      `json:"cpuUsage"`
	MemoryBytes         int64                        `json:"memoryBytes"`
	MemoryUsage         float64                      `json:"memoryUsage"`
	AgentCPUUsageMillis *int64                       `json:"agentCpuUsageMillis,omitempty"`
	AgentMemoryBytes    *int64                       `json:"agentMemoryBytes,omitempty"`
	DiskMetrics         []DeploymentTargetDiskMetric `json:"diskMetrics,omitempty"`
}

type AgentDeploymentResourceMetricsRequest struct {
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
}

type DeploymentTargetDiskMetric struct {
	Device     string `json:"device"`
	Path       string `json:"path"`
	FsType     string `json:"fsType"`
	BytesTotal int64  `json:"bytesTotal"`
	BytesUsed  int64  `json:"bytesUsed"`
}
