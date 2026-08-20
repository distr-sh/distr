package types

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentMetrics struct {
	ID           uuid.UUID                  `db:"id"`
	CreatedAt    time.Time                  `db:"created_at"`
	DeploymentID uuid.UUID                  `db:"deployment_id"`
	Resources    []DeploymentResourceMetric `db:"resources"`
}

type DeploymentResourceMetric struct {
	Resource         string
	Container        string
	CPUUsageMillis   int64
	MemoryBytes      int64
	CPULimitMillis   *int64
	MemoryLimitBytes *int64
}
