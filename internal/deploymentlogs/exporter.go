package deploymentlogs

import (
	"context"

	"github.com/distr-sh/distr/api"
)

type Exporter interface {
	ExportDeploymentLogs(ctx context.Context, records []api.DeploymentLogRecord) error
}
