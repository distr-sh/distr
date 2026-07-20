package deploymenttargetlogs

import (
	"errors"

	"github.com/distr-sh/distr/api"
)

// ErrRecordsRejected indicates that the server permanently rejected a batch of
// log records (e.g. an HTTP 400 response). Such records must not be retried, as
// doing so would fail forever and block all newer logs.
var ErrRecordsRejected = errors.New("log records rejected by server")

type Exporter interface {
	ExportDeploymentTargetLogs(records ...api.DeploymentTargetLogRecord) error
}

type Syncer interface {
	Sync() error
}
