package api

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentRevisionStatus struct {
	CreatedAt            time.Time `json:"createdAt"`
	DeploymentRevisionID uuid.UUID `json:"deploymentRevisionId"`
	Type                 string    `json:"type"`
	Message              string    `json:"message"`
}
