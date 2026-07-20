package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/logstore"
)

func DeploymentTargetLogRecordToAPI(record logstore.DeploymentTargetLogRecord) api.DeploymentTargetLogRecord {
	return api.DeploymentTargetLogRecord{
		ID:        record.ID,
		Timestamp: record.Timestamp,
		Severity:  record.Severity,
		Body:      record.Body,
	}
}
