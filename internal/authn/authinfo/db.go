package authinfo

import (
	"context"
	"errors"

	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/types"
)

type DbAuthInfo struct {
	AuthInfo
	user        *types.UserAccount
	org         *types.OrganizationWithBranding
	customerOrg *types.CustomerOrganization
}

func (a DbAuthInfo) CurrentUser() *types.UserAccount {
	return a.user
}

func (a DbAuthInfo) CurrentCustomerOrg() *types.CustomerOrganization {
	return a.customerOrg
}

func (a DbAuthInfo) CurrentOrg() *types.Organization {
	if a.org == nil {
		return nil
	}
	return &a.org.Organization
}

func (a DbAuthInfo) CurrentOrgWithBranding() *types.OrganizationWithBranding {
	return a.org
}

func DbAuthenticator() authn.Authenticator[AuthInfo, AuthInfoWithUserAndOrganization] {
	fn := func(ctx context.Context, a AuthInfo) (AuthInfoWithUserAndOrganization, error) {
		if a.CurrentOrgID() != nil {
			// Super admins: skip membership check, just verify user and org exist
			if a.IsSuperAdmin() {
				user, err := db.GetUserAccountByID(ctx, a.CurrentUserID())
				if errors.Is(err, distrerr.ErrNotFound) {
					return nil, authn.ErrBadAuthentication
				} else if err != nil {
					return nil, err
				} else if !user.IsSuperAdmin {
					// A token issued before the flag was revoked still carries the claim.
					return nil, authn.ErrBadAuthentication
				}
				org, err := db.GetOrganizationWithBranding(ctx, *a.CurrentOrgID())
				if errors.Is(err, distrerr.ErrNotFound) {
					return nil, authn.ErrBadAuthentication
				} else if err != nil {
					return nil, err
				}
				portal, customerOrg, foreignContext, err := sessionPortal(a, Portal{}, nil,
					func(own, requested Portal) (*types.CustomerOrganization, error) {
						return EnterPortal(ctx, org.ID, own, true, requested)
					},
				)
				if err != nil {
					return nil, err
				}
				return &DbAuthInfo{
					AuthInfo: &SimpleAuthInfo{
						userID:                 a.CurrentUserID(),
						userEmail:              a.CurrentUserEmail(),
						organizationID:         a.CurrentOrgID(),
						customerOrganizationID: portal.CustomerOrganizationID,
						partnerOrganizationID:  portal.PartnerOrganizationID,
						emailVerified:          a.CurrentUserEmailVerified(),
						tokenScope:             a.TokenScope(),
						isAccessToken:          a.IsAccessToken(),
						isCustomOIDCSession:    a.IsCustomOIDCSession(),
						userRole:               nil, // Super admins don't have a role
						isSuperAdmin:           true,
						isForeignContext:       foreignContext,
						rawToken:               a.Token(),
					},
					user:        user,
					org:         org,
					customerOrg: customerOrg,
				}, nil
			}
			// Regular users: require org membership and validate role
			if a.CurrentUserRole() != nil {
				requested := requestedPortal(a)
				if m, err := db.GetOrganizationMembership(
					ctx,
					a.CurrentUserID(),
					*a.CurrentOrgID(),
					requested.CustomerOrganizationID,
					requested.PartnerOrganizationID,
				); errors.Is(err, distrerr.ErrNotFound) {
					return nil, authn.ErrBadAuthentication
				} else if err != nil {
					return nil, err
				} else if a.CurrentUserRole().GreaterThan(m.User.UserRole) {
					// The role carried by the credential must never exceed the
					// user's current role in the organization. Equal-or-below is
					// fine (e.g. a PAT scoped to a lower role); above means the
					// user was demoted after the credential was issued.
					return nil, authn.ErrBadAuthentication
				} else if portal, customerOrg, foreignContext, err := sessionPortal(
					a,
					Portal{
						CustomerOrganizationID: m.User.CustomerOrganizationID,
						PartnerOrganizationID:  m.User.PartnerOrganizationID,
					},
					m.CustomerOrganization,
					func(own, requested Portal) (*types.CustomerOrganization, error) {
						return enterLoadedPortal(
							own, false, requested, m.RequestedCustomerOrganization, m.RequestedPartnerOrganizationExists,
						)
					},
				); err != nil {
					return nil, err
				} else {
					return &DbAuthInfo{
						AuthInfo: &SimpleAuthInfo{
							userID:                 a.CurrentUserID(),
							userEmail:              a.CurrentUserEmail(),
							organizationID:         a.CurrentOrgID(),
							customerOrganizationID: portal.CustomerOrganizationID,
							partnerOrganizationID:  portal.PartnerOrganizationID,
							emailVerified:          a.CurrentUserEmailVerified(),
							tokenScope:             a.TokenScope(),
							isAccessToken:          a.IsAccessToken(),
							isCustomOIDCSession:    a.IsCustomOIDCSession(),
							userRole:               a.CurrentUserRole(),
							isSuperAdmin:           false,
							isForeignContext:       foreignContext,
							rawToken:               a.Token(),
						},
						user:        new(m.User.AsUserAccount()),
						org:         &m.Organization,
						customerOrg: customerOrg,
					}, nil
				}
			}
			return nil, authn.ErrBadAuthentication
		} else {
			// some special tokens like password reset don't have an organization ID
			if u, err := db.GetUserAccountByID(ctx, a.CurrentUserID()); errors.Is(err, distrerr.ErrNotFound) {
				return nil, authn.ErrBadAuthentication
			} else if err != nil {
				return nil, err
			} else {
				return &DbAuthInfo{AuthInfo: a, user: u}, nil
			}
		}
	}
	return authn.AuthenticatorFunc[AuthInfo, AuthInfoWithUserAndOrganization](fn)
}

// sessionPortal returns the portal a session operates in, the customer organization of that portal and
// whether that portal is a foreign context, one the session operates in without being a member. A personal
// access token always operates in its membership's portal: only a login session switches, through the
// switch-context endpoint. enter decides whether the session may operate in a requested portal that is not
// its own.
func sessionPortal(
	a AuthInfo,
	own Portal,
	ownCustomerOrg *types.CustomerOrganization,
	enter func(own, requested Portal) (*types.CustomerOrganization, error),
) (Portal, *types.CustomerOrganization, bool, error) {
	requested := requestedPortal(a)
	if IsOwnPortal(own, requested) {
		return own, ownCustomerOrg, false, nil
	}
	customerOrg, err := enter(own, requested)
	if errors.Is(err, ErrPortalForbidden) {
		return Portal{}, nil, false, authn.ErrBadAuthentication
	} else if err != nil {
		return Portal{}, nil, false, err
	}
	return requested, customerOrg, true, nil
}

func requestedPortal(a AuthInfo) Portal {
	if a.IsAccessToken() {
		return Portal{}
	}
	return Portal{CustomerOrganizationID: a.CurrentCustomerOrgID(), PartnerOrganizationID: a.CurrentPartnerOrgID()}
}

type controllerDBAuthInfo struct {
	AuthInfo
	org *types.OrganizationWithBranding
}

func (a controllerDBAuthInfo) CurrentOrg() *types.Organization {
	if a.org == nil {
		return nil
	}
	return &a.org.Organization
}

func (a controllerDBAuthInfo) CurrentOrgWithBranding() *types.OrganizationWithBranding {
	return a.org
}

func ControllerDbAuthenticator() authn.Authenticator[ControllerAuthInfo, AuthInfoWithOrganization] {
	fn := func(ctx context.Context, a ControllerAuthInfo) (AuthInfoWithOrganization, error) {
		customer, org, err := db.GetCustomerAndOrgForDeploymentTarget(ctx, a.CurrentDeploymentTargetID())
		if errors.Is(err, distrerr.ErrNotFound) {
			return nil, authn.ErrBadAuthentication
		} else if err != nil {
			return nil, err
		}
		deploymentTargetID := a.CurrentDeploymentTargetID()
		info := &SimpleAuthInfo{
			organizationID:     &org.ID,
			deploymentTargetID: &deploymentTargetID,
			rawToken:           a.Token(),
		}
		if customer != nil {
			info.customerOrganizationID = &customer.ID
		}
		return &controllerDBAuthInfo{AuthInfo: info, org: org}, nil
	}
	return authn.AuthenticatorFunc[ControllerAuthInfo, AuthInfoWithOrganization](fn)
}

func DropUser() authn.Authenticator[AuthInfoWithUserAndOrganization, AuthInfoWithOrganization] {
	fn := func(ctx context.Context, a AuthInfoWithUserAndOrganization) (AuthInfoWithOrganization, error) {
		return a, nil
	}
	return authn.AuthenticatorFunc[AuthInfoWithUserAndOrganization, AuthInfoWithOrganization](fn)
}
