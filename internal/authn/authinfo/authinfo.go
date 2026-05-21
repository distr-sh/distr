package authinfo

import (
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type AuthInfo interface {
	CurrentUserID() uuid.UUID
	CurrentUserEmail() string
	CurrentAccountRole() *types.AccountRole
	CurrentOrgID() *uuid.UUID
	CurrentCustomerOrgID() *uuid.UUID
	CurrentUserEmailVerified() bool
	IsSuperAdmin() bool
	// CurrentServiceAccountID returns the ID of the calling service account, or nil when the
	// caller is a human user. Service-account-backed AuthInfo has CurrentUserID() == uuid.Nil and
	// CurrentUserEmail() == "".
	CurrentServiceAccountID() *uuid.UUID
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
