package authinfo

import (
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type AuthInfo interface {
	CurrentUserID() uuid.UUID
	CurrentUserEmail() string
	CurrentUserRole() *types.UserRole
	CurrentOrgID() *uuid.UUID
	CurrentCustomerOrgID() *uuid.UUID
	CurrentUserEmailVerified() bool
	IsSuperAdmin() bool
	Token() any
}

type AgentAuthInfo interface {
	CurrentDeploymentTargetID() uuid.UUID
	CurrentOrgID() uuid.UUID
	Token() any
}

type AuthInfoWithOrganization interface {
	AuthInfo
	CurrentOrg() *types.Organization
}

type AuthInfoWithUserAndOrganization interface {
	AuthInfoWithOrganization
	CurrentUser() *types.UserAccount
}
