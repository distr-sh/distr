package deploymentlogs

import (
	"context"
	"errors"

	"github.com/distr-sh/distr/api"
)

// ErrRecordsRejected indicates that the server permanently rejected the records
// (an HTTP 4xx other than 408 and 429), e.g. because an entry is malformed or the batch
// is too large. Such records must be dropped instead of retried: retrying would fail
// forever and block all newer logs for the deployment.
// Loki still ingests the valid entries of a batch even when it returns 400.
var ErrRecordsRejected = errors.New("log records rejected by server")

type Exporter interface {
	ExportDeploymentLogs(ctx context.Context, records []api.DeploymentLogRecord) error
}
