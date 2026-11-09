package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type ArtifactUpstreamAuth struct {
	Type     types.UpstreamAuthType `json:"type"`
	Username *string                `json:"username,omitempty"`
	Password *string                `json:"password,omitempty"`
}

type CreateArtifactRequest struct {
	Name         string                `json:"name"`
	UpstreamURL  *string               `json:"upstreamUrl,omitempty"`
	UpstreamAuth *ArtifactUpstreamAuth `json:"upstreamAuth,omitempty"`
}

// PatchArtifactRequest supports partial updates: omitted fields are left unchanged.
// For auth, explicit null clears the credentials.
type PatchArtifactRequest struct {
	UpstreamURL        *string                        `json:"upstreamUrl,omitempty"`
	Auth               Nullable[ArtifactUpstreamAuth] `json:"auth"`
	Public             *bool                          `json:"public,omitempty"`
	Featured           *bool                          `json:"featured,omitempty"`
	VersioningStrategy *types.VersioningStrategy      `json:"versioningStrategy,omitempty"`
	// DeleteNonSemverTags confirms that switching a pull-through artifact to semver deletes its tags that are
	// not SemVer versions.
	DeleteNonSemverTags bool `json:"deleteNonSemverTags,omitempty"`
}

type ArtifactResponse struct {
	types.ArtifactWithTaggedVersion
	ImageUrl           *string                  `json:"imageUrl,omitempty"`
	TitleImageUrl      *string                  `json:"titleImageUrl,omitempty"`
	Featured           bool                     `json:"featured"`
	VersioningStrategy types.VersioningStrategy `json:"versioningStrategy"`
}

type SyncArtifactResponse struct {
	ArtifactResponse
	// SkippedTags are the upstream tags the sync skipped because the versioning strategy of the artifact does not
	// allow them.
	SkippedTags []string `json:"skippedTags,omitempty"`
}

type DeleteArtifactTagResponse struct {
	ArtifactDeleted bool              `json:"artifactDeleted"`
	Artifact        *ArtifactResponse `json:"artifact,omitempty"`
}

type ArtifactsResponse struct {
	types.ArtifactWithDownloads
	ImageUrl           *string                  `json:"imageUrl,omitempty"`
	TitleImageUrl      *string                  `json:"titleImageUrl,omitempty"`
	Featured           bool                     `json:"featured"`
	VersioningStrategy types.VersioningStrategy `json:"versioningStrategy"`
	// LastPull is only set for a customer.
	LastPull *ArtifactLastPull `json:"lastPull,omitempty"`
}

// ArtifactLastPull is the most recent pull of a release tag of the artifact by the customer organization.
type ArtifactLastPull struct {
	VersionName          string    `json:"versionName"`
	CreatedAt            time.Time `json:"createdAt"`
	PulledByName         *string   `json:"pulledByName,omitempty"`
	PulledByImageUrl     *string   `json:"pulledByImageUrl,omitempty"`
	DeploymentTargetName *string   `json:"deploymentTargetName,omitempty"`
}

type ArtifactVersionPullResponse struct {
	CreatedAt                time.Time             `json:"createdAt"`
	RemoteAddress            *string               `json:"remoteAddress,omitempty"`
	Anonymous                bool                  `json:"anonymous"`
	UserAccountName          *string               `json:"userAccountName,omitempty"`
	UserAccountEmail         *string               `json:"userAccountEmail,omitempty"`
	CustomerOrganizationName *string               `json:"customerOrganizationName,omitempty"`
	DeploymentTargetID       *uuid.UUID            `json:"deploymentTargetId,omitempty"`
	DeploymentTargetName     *string               `json:"deploymentTargetName,omitempty"`
	Artifact                 types.Artifact        `json:"artifact"`
	ArtifactVersion          types.ArtifactVersion `json:"artifactVersion"`
}

type ArtifactPullFilterOption struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ArtifactVersionPullFilterOptions struct {
	CustomerOrganizations []ArtifactPullFilterOption `json:"customerOrganizations"`
	UserAccounts          []ArtifactPullFilterOption `json:"userAccounts"`
	RemoteAddresses       []string                   `json:"remoteAddresses"`
	Artifacts             []ArtifactPullFilterOption `json:"artifacts"`
	DeploymentTargets     []ArtifactPullFilterOption `json:"deploymentTargets"`
}
