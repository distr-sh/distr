package types

import (
	"time"

	"github.com/google/uuid"
)

type ArtifactVersionPull struct {
	CreatedAt     time.Time
	RemoteAddress *string
	// Anonymous says the pull came without credentials, which only a public artifact allows.
	Anonymous bool
	// UserAccount is nil when an agent pulled the artifact with its own token, in which case
	// DeploymentTarget identifies it instead, and when the pull was anonymous.
	UserAccount          *UserAccount
	CustomerOrganization *CustomerOrganization
	DeploymentTarget     *ArtifactPullDeploymentTarget
	Artifact             Artifact
	ArtifactVersion      ArtifactVersion
}

type ArtifactPullDeploymentTarget struct {
	ID   uuid.UUID
	Name string
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
	DeploymentTargets     []FilterOption
}

type ArtifactVersionPullFilter struct {
	OrgID                  uuid.UUID
	PartnerOrganizationID  *uuid.UUID
	Before                 time.Time
	After                  time.Time
	Count                  int
	CustomerOrganizationID *uuid.UUID
	UserAccountID          *uuid.UUID
	Anonymous              bool
	RemoteAddress          *string
	ArtifactID             *uuid.UUID
	ArtifactVersionID      *uuid.UUID
	DeploymentTargetID     *uuid.UUID
}
