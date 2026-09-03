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
	// CurrentDeploymentTargetID returns the deployment target an agent token was issued for, and nil for
	// every credential that does not belong to an agent.
	CurrentDeploymentTargetID() *uuid.UUID
	CurrentUserEmailVerified() bool
	// TokenScope returns the purpose a special, unscoped token was minted for, or the empty
	// scope for regular login tokens, PATs and agent tokens.
	TokenScope() authjwt.TokenScope
	// OrganizationScoped reports whether the credential is confined to the organization it was issued
	// for and is not proof that the account's owner is present: a PAT, which is created for one
	// organization, or a session authenticated by an organization's own identity provider, which the
	// organization controls rather than the account's owner. Such a credential must not switch to
	// another organization, learn about the others the account belongs to, create one, or change the
	// account's sign-in methods — the last one because a password or an email address it could set
	// would let it escape all of the others.
	OrganizationScoped() bool
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
	CurrentOrgWithBranding() *types.OrganizationWithBranding
}

type AuthInfoWithUserAndOrganization interface {
	AuthInfoWithOrganization
	CurrentUser() *types.UserAccount
}
