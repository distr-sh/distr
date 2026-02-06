package types

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentRevisionStatus struct {
	ID                   uuid.UUID            `db:"id" json:"id"`
	CreatedAt            time.Time            `db:"created_at" json:"createdAt"`
	DeploymentRevisionID uuid.UUID            `db:"deployment_revision_id" json:"deploymentRevisionId"`
	Type                 DeploymentStatusType `db:"type" json:"type"`
	Message              string               `db:"message" json:"message"`
}

const deploymentRevisionStatusStaleDuration = 1 * time.Minute

func (s *DeploymentRevisionStatus) IsStale() bool {
	return s != nil && time.Since(s.CreatedAt) > deploymentRevisionStatusStaleDuration
}
