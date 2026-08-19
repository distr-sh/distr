package types

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentMetrics struct {
	ID           uuid.UUID                  `db:"id"`
	CreatedAt    time.Time                  `db:"created_at"`
	DeploymentID uuid.UUID                  `db:"deployment_id"`
	Workloads    []DeploymentWorkloadMetric `db:"workloads"`
}

type DeploymentWorkloadMetric struct {
	Workload         string
	Name             string
	CPUUsageMillis   int64
	MemoryBytes      int64
	CPULimitMillis   *int64
	MemoryLimitBytes *int64
}
