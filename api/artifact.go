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
	UpstreamURL *string                        `json:"upstreamUrl,omitempty"`
	Auth        Nullable[ArtifactUpstreamAuth] `json:"auth"`
	Public      *bool                          `json:"public,omitempty"`
}

type ArtifactResponse struct {
	types.ArtifactWithTaggedVersion
	ImageUrl *string `json:"imageUrl,omitempty"`
}

type ArtifactsResponse struct {
	types.ArtifactWithDownloads
	ImageUrl *string `json:"imageUrl,omitempty"`
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
