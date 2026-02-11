package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

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
	UserAccountName          *string               `json:"userAccountName,omitempty"`
	UserAccountEmail         *string               `json:"userAccountEmail,omitempty"`
	CustomerOrganizationName *string               `json:"customerOrganizationName,omitempty"`
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
}
