package api

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentLogRecord struct {
	DeploymentID         uuid.UUID `json:"deploymentId"`
	DeploymentRevisionID uuid.UUID `json:"deploymentRevisionId"`
	Resource             string    `json:"resource"`
	Timestamp            time.Time `json:"timestamp"`
	Severity             string    `json:"severity"`
	Body                 string    `json:"body"`
}

type DeploymentLogRecordResourcesResponse struct {
	Active   []string `json:"active"`
	Archived []string `json:"archived"`
}
