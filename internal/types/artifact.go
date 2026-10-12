package types

import (
	"regexp"
	"slices"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
)

// NonReleaseTagPattern matches the tags clients push for signatures and attestations, the OCI referrers
// fallback tag "sha256-<digest>" and the cosign tags "sha256-<digest>.sig", ".att", ".sbom" and ".cosign",
// so that these are not mistaken for releases. It is used both in Go and in PostgreSQL, so it must stay
// within the syntax both understand.
const NonReleaseTagPattern = `^sha256-[0-9a-f]{64}(\.(sig|att|sbom|cosign))?$`

var nonReleaseTagRegexp = regexp.MustCompile(NonReleaseTagPattern)

func IsReleaseTag(name string) bool {
	return !nonReleaseTagRegexp.MatchString(name)
}

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
	ID                       uuid.UUID         `db:"id" json:"id"`
	CreatedAt                time.Time         `db:"created_at" json:"createdAt"`
	OrganizationID           uuid.UUID         `db:"organization_id" json:"-"`
	Name                     string            `db:"name" json:"name"`
	ImageID                  *uuid.UUID        `db:"image_id" json:"-"`
	TitleImageID             *uuid.UUID        `db:"title_image_id" json:"-"`
	Public                   bool              `db:"public" json:"public"`
	HiddenInCustomerOverview bool              `db:"hidden_in_customer_overview" json:"-"`
	UpstreamURL              *string           `db:"upstream_url" json:"upstreamUrl,omitempty"`
	LastSyncedAt             *time.Time        `db:"last_synced_at" json:"lastSyncedAt,omitempty"`
	LastSyncError            *string           `db:"last_sync_error" json:"lastSyncError,omitempty"`
	UpstreamAuthType         *UpstreamAuthType `db:"upstream_auth_type" json:"upstreamAuthType,omitempty"`
	UpstreamUsername         *dbcrypto.String  `db:"upstream_username" json:"-"`
	UpstreamPassword         *dbcrypto.String  `db:"upstream_password" json:"-"`
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

// CustomerArtifactPull is the most recent pull of a release tag of an artifact by a customer organization.
type CustomerArtifactPull struct {
	ArtifactID           uuid.UUID  `db:"artifact_id"`
	VersionName          string     `db:"version_name"`
	CreatedAt            time.Time  `db:"created_at"`
	UserName             *string    `db:"user_name"`
	UserImageID          *uuid.UUID `db:"user_image_id"`
	DeploymentTargetName *string    `db:"deployment_target_name"`
}

// LatestArtifactTag returns the newest of the versions along with the tag to show for it. Signatures and
// tags that are not release tags are skipped. Versions are ordered by their highest SemVer tag, or by
// creation as soon as one of the release tags is not a SemVer version. The tag shown is the highest
// SemVer tag of the version, or its first release tag when it has none.
func LatestArtifactTag(versions []TaggedArtifactVersion) (*TaggedArtifactVersion, string) {
	type candidate struct {
		version *TaggedArtifactVersion
		semver  *semver.Version
		tag     string
	}
	var candidates []candidate
	bySemver := true
	for i := range versions {
		if versions[i].InferredType == ManifestTypeSignature {
			continue
		}
		var c *candidate
		for _, tag := range versions[i].Tags {
			if !IsReleaseTag(tag.Name) {
				continue
			} else if c == nil {
				c = &candidate{version: &versions[i], tag: tag.Name}
			}
			if parsed, err := semver.NewVersion(tag.Name); err != nil {
				bySemver = false
			} else if c.semver == nil || parsed.GreaterThan(c.semver) {
				c.semver = parsed
				c.tag = tag.Name
			}
		}
		if c != nil {
			candidates = append(candidates, *c)
		}
	}

	var latest *candidate
	for i := range candidates {
		if c := &candidates[i]; latest == nil ||
			(bySemver && c.semver.GreaterThan(latest.semver)) ||
			(!bySemver && c.version.CreatedAt.After(latest.version.CreatedAt)) {
			latest = c
		}
	}
	if latest == nil {
		return nil, ""
	}
	return latest.version, latest.tag
}

// ArtifactPullIsLatest reports whether the pulled tag points to latest. A tag that none of the versions
// carries anymore counts as latest, since there is nothing left to compare it with.
func ArtifactPullIsLatest(versions []TaggedArtifactVersion, latest *TaggedArtifactVersion, pulledTag string) bool {
	hasTag := func(v TaggedArtifactVersion) bool {
		return slices.ContainsFunc(v.Tags, func(t ArtifactVersionTag) bool { return t.Name == pulledTag })
	}
	if latest != nil && hasTag(*latest) {
		return true
	}
	return !slices.ContainsFunc(versions, hasTag)
}
