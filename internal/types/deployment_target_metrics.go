package types

import (
	"github.com/google/uuid"
)

type DeploymentTargetMetrics struct {
	ID                 uuid.UUID                    `db:"id"`
	DeploymentTargetID uuid.UUID                    `db:"deployment_target_id"`
	CPUCoresMillis     int64                        `db:"cpu_cores_millis"`
	CPUUsage           float64                      `db:"cpu_usage"`
	MemoryBytes        int64                        `db:"memory_bytes"`
	MemoryUsage        float64                      `db:"memory_usage"`
	DiskMetrics        []DeploymentTargetDiskMetric `db:"disk_metrics"`
}

type DeploymentTargetDiskMetric struct {
	Device     string
	Path       string
	FsType     string
	BytesTotal int64
	BytesUsed  int64
}

func (m DeploymentTargetDiskMetric) Usage() float64 {
	return float64(m.BytesUsed) / float64(m.BytesTotal)
}
