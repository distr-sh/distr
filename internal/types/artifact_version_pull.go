package types

import (
	"time"

	"github.com/google/uuid"
)

type ArtifactVersionPull struct {
	CreatedAt            time.Time             `json:"createdAt"`
	RemoteAddress        *string               `json:"remoteAddress,omitempty"`
	UserAccount          *UserAccount          `json:"userAccount,omitempty"`
	CustomerOrganization *CustomerOrganization `json:"customerOrganization,omitempty"`
	Artifact             Artifact              `json:"artifact"`
	ArtifactVersion      ArtifactVersion       `json:"artifactVersion"`
}

type FilterOption struct {
	ID   uuid.UUID
	Name string
}

type ArtifactVersionPullFilterOptions struct {
	CustomerOrganizations []FilterOption
	UserAccounts          []FilterOption
	RemoteAddresses       []string
	Artifacts             []FilterOption
}

type ArtifactVersionPullFilter struct {
	OrgID                  uuid.UUID
	Before                 time.Time
	After                  time.Time
	Count                  int
	CustomerOrganizationID *uuid.UUID
	UserAccountID          *uuid.UUID
	RemoteAddress          *string
	ArtifactID             *uuid.UUID
	ArtifactVersionID      *uuid.UUID
}
