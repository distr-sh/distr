package deploymentlogs

import (
	"context"
	"errors"

	"github.com/distr-sh/distr/api"
)

// ErrRecordsRejected indicates that the log store permanently rejected the records
// (HTTP 400), e.g. because an entry is malformed. Such records must be dropped instead
// of retried: retrying would fail forever and block all newer logs for the deployment.
// Loki still ingests the valid entries of a batch even when it returns 400.
var ErrRecordsRejected = errors.New("log records rejected by server")

type Exporter interface {
	ExportDeploymentLogs(ctx context.Context, records []api.DeploymentLogRecord) error
}
