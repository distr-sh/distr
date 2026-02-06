package types

import (
	"time"

	"github.com/google/uuid"
)

type ManifestType string

const (
	ManifestTypeGeneric        ManifestType = "generic"
	ManifestTypeContainerImage ManifestType = "container-image"
	ManifestTypeHelmChart      ManifestType = "helm-chart"
)

type Artifact struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	CreatedAt      time.Time  `db:"created_at" json:"createdAt"`
	OrganizationID uuid.UUID  `db:"organization_id" json:"-"`
	Name           string     `db:"name" json:"name"`
	ImageID        *uuid.UUID `db:"image_id" json:"-"`
}

type DownloadMetrics struct {
	DownloadsTotal                    int         `db:"downloads_total" json:"downloadsTotal"`
	DownloadedByCount                 int         `db:"downloaded_by_count" json:"downloadedByCount"`
	DownloadedByUsers                 []uuid.UUID `db:"downloaded_by_users" json:"downloadedByUsers,omitempty"`
	DownloadedByCustomerOrganizations []uuid.UUID `db:"downloaded_by_customer_organizations" json:"downloadedByCustomerOrganizations,omitempty"` //nolint:lll
}

type ArtifactVersionTag struct {
	ID   uuid.UUID `db:"id" json:"id"`
	Name string    `db:"name" json:"name"`

	Downloads DownloadMetrics `json:"downloads"`
}

type TaggedArtifactVersion struct {
	ID                  uuid.UUID            `db:"id" json:"id"`
	CreatedAt           time.Time            `db:"created_at" json:"createdAt"`
	Digest              string               `db:"manifest_blob_digest" json:"digest"`
	ManifestContentType string               `db:"manifest_content_type" json:"manifestContentType"`
	ManifestData        []byte               `db:"manifest_data" json:"-"`
	Tags                []ArtifactVersionTag `db:"tags" json:"tags"`
	Size                int64                `db:"size" json:"size"`

	DownloadsTotal                    int         `db:"downloads_total" json:"downloadsTotal"`
	DownloadedByCount                 int         `db:"downloaded_by_count" json:"downloadedByCount"`
	DownloadedByUsers                 []uuid.UUID `db:"downloaded_by_users" json:"downloadedByUsers,omitempty"`
	DownloadedByCustomerOrganizations []uuid.UUID `db:"downloaded_by_customer_organizations" json:"downloadedByCustomerOrganizations,omitempty"` //nolint:lll

	InferredType ManifestType `db:"-" json:"inferredType"`
}

type ArtifactWithDownloads struct {
	Artifact
	OrganizationSlug                  string      `db:"organization_slug" json:"-"`
	DownloadsTotal                    int         `db:"downloads_total" json:"downloadsTotal"`
	DownloadedByCount                 int         `db:"downloaded_by_count" json:"downloadedByCount"`
	DownloadedByUsers                 []uuid.UUID `db:"downloaded_by_users" json:"downloadedByUsers,omitempty"`
	DownloadedByCustomerOrganizations []uuid.UUID `db:"downloaded_by_customer_organizations" json:"downloadedByCustomerOrganizations,omitempty"` //nolint:lll
}

type ArtifactWithTaggedVersion struct {
	ArtifactWithDownloads
	Versions []TaggedArtifactVersion `db:"versions" json:"versions,omitempty"`
}
