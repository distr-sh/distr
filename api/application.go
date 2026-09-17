package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type ApplicationResponse struct {
	types.Application
	ImageUrl *string `json:"imageUrl,omitempty"`
}

type CreateApplicationRequest struct {
	Name string               `json:"name"`
	Type types.DeploymentType `json:"type"`
	// VersioningStrategy defaults to chronological when absent, which imposes no requirement on
	// version names and so cannot break a client written before the field existed.
	VersioningStrategy    types.VersioningStrategy `json:"versioningStrategy,omitempty"`
	AllowAutomaticUpdates bool                     `json:"allowAutomaticUpdates,omitempty"`
}

type UpdateApplicationRequest struct {
	Name                  string                   `json:"name"`
	VersioningStrategy    types.VersioningStrategy `json:"versioningStrategy,omitempty"`
	AllowAutomaticUpdates bool                     `json:"allowAutomaticUpdates,omitempty"`
}

type PatchApplicationRequest struct {
	Name                  *string                          `json:"name,omitempty"`
	VersioningStrategy    *types.VersioningStrategy        `json:"versioningStrategy,omitempty"`
	AllowAutomaticUpdates *bool                            `json:"allowAutomaticUpdates,omitempty"`
	Versions              []PatchApplicationVersionRequest `json:"versions,omitempty"`
}

type PatchApplicationVersionRequest struct {
	ID         uuid.UUID  `json:"id"`
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
}
