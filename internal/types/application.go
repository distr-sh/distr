package types

import (
	"time"

	"github.com/google/uuid"
)

// Field order matters here: a nested application row is a Postgres composite that pgx scans
// positionally, ignoring the db tags, so it must match the column order of applicationOutputExpr.
// Versions stays last because the queries selecting a nested row leave it out.
type Application struct {
	ID                    uuid.UUID            `db:"id" json:"id"`
	CreatedAt             time.Time            `db:"created_at" json:"createdAt"`
	OrganizationID        uuid.UUID            `db:"organization_id" json:"-"`
	Name                  string               `db:"name" json:"name"`
	Type                  DeploymentType       `db:"type" json:"type"`
	ImageID               *uuid.UUID           `db:"image_id" json:"imageId,omitempty"`
	VersioningStrategy    VersioningStrategy   `db:"versioning_strategy" json:"versioningStrategy"`
	AllowAutomaticUpdates bool                 `db:"allow_automatic_updates" json:"allowAutomaticUpdates"`
	Versions              []ApplicationVersion `db:"versions" json:"versions"`
}
