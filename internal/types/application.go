package types

import (
	"time"

	"github.com/google/uuid"
)

type Application struct {
	ID             uuid.UUID            `db:"id" json:"id"`
	CreatedAt      time.Time            `db:"created_at" json:"createdAt"`
	OrganizationID uuid.UUID            `db:"organization_id" json:"-"`
	Name           string               `db:"name" json:"name"`
	Type           DeploymentType       `db:"type" json:"type"`
	ImageID        *uuid.UUID           `db:"image_id" json:"imageId"`
	Versions       []ApplicationVersion `db:"versions" json:"versions"`
}
