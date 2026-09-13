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

// DeploymentResourceMetric is decoded from a Postgres row() composite, which pgx maps to these
// fields by position rather than by name. The field order must stay in lockstep with the column
// order of the row() expression in deploymentMetricsOutputExpr.
type DeploymentResourceMetric struct {
	Resource         string
	Container        string
	CPUUsageMillis   int64
	MemoryBytes      int64
	CPULimitMillis   *int64
	MemoryLimitBytes *int64
	LogBytes         *int64
}
