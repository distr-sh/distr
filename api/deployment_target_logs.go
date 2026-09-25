package api

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentTargetLogRecordRequest struct {
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Body      string    `json:"body" trim:"-"`
}

type DeploymentTargetLogRecord struct {
	ID        uuid.UUID `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"`
	Body      string    `json:"body"`
}
