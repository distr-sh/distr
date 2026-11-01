package types

import (
	"time"

	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
)

// NonReleaseTagPattern matches the tags clients push for signatures and attestations, the OCI referrers
// fallback tag "sha256-<digest>" and the cosign tags "sha256-<digest>.sig", ".att", ".sbom" and ".cosign",
// so that these are not mistaken for releases. It is a PostgreSQL regular expression.
const NonReleaseTagPattern = `^sha256-[0-9a-f]{64}(\.(sig|att|sbom|cosign))?$`

type ManifestType string

const (
	ManifestTypeGeneric        ManifestType = "generic"
	ManifestTypeContainerImage ManifestType = "container-image"
	ManifestTypeHelmChart      ManifestType = "helm-chart"
	ManifestTypeSignature      ManifestType = "signature"
)

type UpstreamAuthType string

const (
	UpstreamAuthTypeBasic  UpstreamAuthType = "basic"
	UpstreamAuthTypeAWSECR UpstreamAuthType = "aws_ecr"
)

type Artifact struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	CreatedAt      time.Time  `db:"created_at" json:"createdAt"`
	OrganizationID uuid.UUID  `db:"organization_id" json:"-"`
	Name           string     `db:"name" json:"name"`
	ImageID        *uuid.UUID `db:"image_id" json:"-"`
	TitleImageID   *uuid.UUID `db:"title_image_id" json:"-"`
	Public         bool       `db:"public" json:"public"`
	// Featured marks one of the main artifacts of the vendor's applications, which the Customer Portal home
	// page shows.
	Featured         bool              `db:"featured" json:"-"`
	UpstreamURL      *string           `db:"upstream_url" json:"upstreamUrl,omitempty"`
	LastSyncedAt     *time.Time        `db:"last_synced_at" json:"lastSyncedAt,omitempty"`
	LastSyncError    *string           `db:"last_sync_error" json:"lastSyncError,omitempty"`
	UpstreamAuthType *UpstreamAuthType `db:"upstream_auth_type" json:"upstreamAuthType,omitempty"`
	UpstreamUsername *dbcrypto.String  `db:"upstream_username" json:"-"`
	UpstreamPassword *dbcrypto.String  `db:"upstream_password" json:"-"`
}

type DownloadMetrics struct {
	DownloadsTotal                         int         `db:"downloads_total" json:"downloadsTotal"`
	DownloadedByUsersCount                 int         `db:"downloaded_by_users_count" json:"downloadedByUsersCount"`
	DownloadedByUsers                      []uuid.UUID `db:"downloaded_by_users" json:"downloadedByUsers,omitempty"`
	DownloadedByCustomerOrganizationsCount int         `db:"downloaded_by_customer_organizations_count" json:"downloadedByCustomerOrganizationsCount"` //nolint:lll
	DownloadedByCustomerOrganizations      []uuid.UUID `db:"downloaded_by_customer_organizations" json:"downloadedByCustomerOrganizations,omitempty"`  //nolint:lll
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

	DownloadMetrics

	ReferrerArtifactTypes []string     `db:"referrer_artifact_types" json:"-"`
	InferredType          ManifestType `db:"-" json:"inferredType"`
}

type ArtifactWithDownloads struct {
	Artifact
	OrganizationSlug string `db:"organization_slug" json:"-"`
	DownloadMetrics
}

type ArtifactWithTaggedVersion struct {
	ArtifactWithDownloads
	Versions []TaggedArtifactVersion `db:"versions" json:"versions"`
}

// SignatureArtifactType is the artifact type of a referrer that marks a version as a signature.
const SignatureArtifactType = "application/vnd.dev.sigstore.bundle.v0.3+json"

// ArtifactReleaseVersion is a tagged version of an artifact with its tag names and whether it is a signature.
type ArtifactReleaseVersion struct {
	ArtifactID uuid.UUID `db:"artifact_id"`
	ID         uuid.UUID `db:"id"`
	CreatedAt  time.Time `db:"created_at"`
	TagNames   []string  `db:"tag_names"`
	Signature  bool      `db:"signature"`
}

// TaggedArtifactVersionsByArtifact converts the versions, grouped by their artifact. A version that is not a
// signature gets the generic type, since its manifest is not loaded.
func TaggedArtifactVersionsByArtifact(versions []ArtifactReleaseVersion) map[uuid.UUID][]TaggedArtifactVersion {
	result := make(map[uuid.UUID][]TaggedArtifactVersion)
	for _, v := range versions {
		tagged := TaggedArtifactVersion{ID: v.ID, CreatedAt: v.CreatedAt, InferredType: ManifestTypeGeneric}
		if v.Signature {
			tagged.InferredType = ManifestTypeSignature
		}
		for _, name := range v.TagNames {
			tagged.Tags = append(tagged.Tags, ArtifactVersionTag{Name: name})
		}
		result[v.ArtifactID] = append(result[v.ArtifactID], tagged)
	}
	return result
}
