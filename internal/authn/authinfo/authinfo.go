package authinfo

import (
	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type AuthInfo interface {
	CurrentUserID() uuid.UUID
	CurrentUserEmail() string
	CurrentUserRole() *types.UserRole
	CurrentOrgID() *uuid.UUID
	CurrentCustomerOrgID() *uuid.UUID
	CurrentPartnerOrgID() *uuid.UUID
	CurrentUserEmailVerified() bool
	// TokenScope returns the purpose a special, unscoped token was minted for, or the empty
	// scope for regular login tokens, PATs and agent tokens.
	TokenScope() authjwt.TokenScope
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
