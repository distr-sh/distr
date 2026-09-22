package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type ApplicationResponse struct {
	ID                    uuid.UUID                    `json:"id"`
	CreatedAt             time.Time                    `json:"createdAt"`
	Name                  string                       `json:"name"`
	Type                  types.DeploymentType         `json:"type"`
	ImageID               *uuid.UUID                   `json:"imageId,omitempty"`
	ImageUrl              *string                      `json:"imageUrl,omitempty"`
	VersioningStrategy    types.VersioningStrategy     `json:"versioningStrategy"`
	AllowAutomaticUpdates bool                         `json:"allowAutomaticUpdates"`
	Versions              []ApplicationVersionResponse `json:"versions"`
}

type ApplicationVersionResponse struct {
	ID            uuid.UUID                  `json:"id"`
	CreatedAt     time.Time                  `json:"createdAt"`
	ArchivedAt    *time.Time                 `json:"archivedAt,omitempty"`
	Name          string                     `json:"name"`
	LinkTemplate  string                     `json:"linkTemplate"`
	ApplicationID uuid.UUID                  `json:"applicationId"`
	ChartType     *types.HelmChartType       `json:"chartType,omitempty"`
	ChartName     *string                    `json:"chartName,omitempty"`
	ChartUrl      *string                    `json:"chartUrl,omitempty"`
	ChartVersion  *string                    `json:"chartVersion,omitempty"`
	CreatedBy     *ApplicationVersionCreator `json:"createdBy,omitempty"`
}

type ApplicationVersionCreator struct {
	ID      uuid.UUID  `json:"id"`
	Name    string     `json:"name,omitempty"`
	Email   string     `json:"email,omitempty"`
	ImageID *uuid.UUID `json:"imageId,omitempty"`
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
	Name               string                   `json:"name"`
	VersioningStrategy types.VersioningStrategy `json:"versioningStrategy,omitempty"`
	// AllowAutomaticUpdates leaves the application's setting alone when it is absent, so that a
	// client written before the field existed cannot turn automatic updates off.
	AllowAutomaticUpdates *bool `json:"allowAutomaticUpdates,omitempty"`
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

type CreateApplicationVersionRequest struct {
	Name         string                              `json:"name"`
	LinkTemplate string                              `json:"linkTemplate,omitempty"`
	ChartType    *types.HelmChartType                `json:"chartType,omitempty"`
	ChartName    *string                             `json:"chartName,omitempty"`
	ChartUrl     *string                             `json:"chartUrl,omitempty"`
	ChartVersion *string                             `json:"chartVersion,omitempty"`
	Resources    []ApplicationVersionResourceRequest `json:"resources,omitempty"`
}

type ApplicationVersionResourceRequest struct {
	Name string `json:"name"`
	// The content is rendered as markdown, where leading whitespace is syntax.
	Content            string `json:"content" trim:"-"`
	VisibleToCustomers bool   `json:"visibleToCustomers"`
}

type UpdateApplicationVersionRequest struct {
	Name       string     `json:"name"`
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
}
