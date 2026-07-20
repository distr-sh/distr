package logstore

import (
	"time"

	"github.com/google/uuid"
)

// DeploymentLogRecord is a single deployment log record as returned by the log store.
// The ID is derived deterministically from the record's content at query time
// (see DeploymentLogRecordID).
type DeploymentLogRecord struct {
	ID                   uuid.UUID
	DeploymentID         uuid.UUID
	DeploymentRevisionID uuid.UUID
	Resource             string
	Timestamp            time.Time
	Severity             string
	Body                 string
}

// DeploymentTargetLogRecord is a single deployment target log record as returned by the
// log store. The ID is derived deterministically from the record's content at query time
// (see DeploymentTargetLogRecordID).
type DeploymentTargetLogRecord struct {
	ID                 uuid.UUID
	DeploymentTargetID uuid.UUID
	Timestamp          time.Time
	Severity           string
	Body               string
}
