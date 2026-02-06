package api

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentRevisionStatus struct {
	ID                   uuid.UUID `json:"id"`
	CreatedAt            time.Time `json:"createdAt"`
	DeploymentRevisionID uuid.UUID `json:"deploymentRevisionId"`
	Type                 string    `json:"type"`
	Message              string    `json:"message"`
}
