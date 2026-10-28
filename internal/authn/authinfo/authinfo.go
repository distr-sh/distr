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
	// CurrentDeploymentTargetID returns the deployment target a controller token was issued for, and nil for
	// every credential that does not belong to a controller.
	CurrentDeploymentTargetID() *uuid.UUID
	CurrentUserEmailVerified() bool
	// TokenScope returns the purpose a special, unscoped token was minted for, or the empty
	// scope for regular login tokens, PATs and controller tokens.
	TokenScope() authjwt.TokenScope
	IsAccessToken() bool
	// IsCustomOIDCSession reports whether an organization's own identity provider authenticated the
	// session, which the organization controls rather than the account's owner.
	IsCustomOIDCSession() bool
	// OrganizationScoped reports whether the credential is confined to the organization it was issued
	// for. Such a credential must not switch to another organization, learn about the others the
	// account belongs to, or create one.
	OrganizationScoped() bool
	IsSuperAdmin() bool
	// IsLens reports whether the request looks through the lens of a customer organization its user is not a member
	// of (see lens.Lens). CurrentCustomerOrgID names that organization, so a handler reading it treats the request
	// like one of a member of it. Only authentication against the database sets this.
	IsLens() bool
	Token() any
}

type ControllerAuthInfo interface {
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
	// CurrentCustomerOrg is nil for every member of the vendor or a partner organization.
	CurrentCustomerOrg() *types.CustomerOrganization
	// LensCustomerOrg is the customer organization a request looks at through a lens, nil outside of a lens. It is
	// never set together with CurrentCustomerOrg.
	LensCustomerOrg() *types.CustomerOrganization
}
