package api

import "github.com/google/uuid"

type DeploymentTargetMetrics struct {
	DeploymentTargetID uuid.UUID                    `json:"deploymentTargetId"`
	CPUCoresMillis     int64                        `json:"cpuCoresMillis"`
	CPUUsage           float64                      `json:"cpuUsage"`
	MemoryBytes        int64                        `json:"memoryBytes"`
	MemoryUsage        float64                      `json:"memoryUsage"`
	DiskMetrics        []DeploymentTargetDiskMetric `json:"diskMetrics,omitempty"`
}

type DeploymentTargetDiskMetric struct {
	Device     string `json:"device"`
	Path       string `json:"path"`
	FsType     string `json:"fsType"`
	BytesTotal int64  `json:"bytesTotal"`
	BytesUsed  int64  `json:"bytesUsed"`
}
