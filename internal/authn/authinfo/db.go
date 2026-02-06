package authinfo

import (
	"context"
	"errors"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
)

type DbAuthInfo struct {
	AuthInfo
	user *types.UserAccount
	org  *types.Organization
}

func (a DbAuthInfo) CurrentUser() *types.UserAccount {
	return a.user
}

func (a DbAuthInfo) CurrentOrg() *types.Organization {
	return a.org
}

func DbAuthenticator() authn.Authenticator[AuthInfo, AuthInfoWithUserAndOrganization] {
	fn := func(ctx context.Context, a AuthInfo) (AuthInfoWithUserAndOrganization, error) {
		if a.CurrentOrgID() != nil {
			// Super admins: skip membership check, just verify user and org exist
			if a.IsSuperAdmin() {
				user, err := db.GetUserAccountByID(ctx, a.CurrentUserID())
				if errors.Is(err, apierrors.ErrNotFound) {
					return nil, authn.ErrBadAuthentication
				} else if err != nil {
					return nil, err
				}
				org, err := db.GetOrganizationByID(ctx, *a.CurrentOrgID())
				if errors.Is(err, apierrors.ErrNotFound) {
					return nil, authn.ErrBadAuthentication
				} else if err != nil {
					return nil, err
				}
				return &DbAuthInfo{
					AuthInfo: &SimpleAuthInfo{
						userID:                 a.CurrentUserID(),
						userEmail:              a.CurrentUserEmail(),
						organizationID:         a.CurrentOrgID(),
						customerOrganizationID: nil,
						emailVerified:          a.CurrentUserEmailVerified(),
						userRole:               nil, // Super admins don't have a role
						isSuperAdmin:           true,
						rawToken:               a.Token(),
					},
					user: user,
					org:  org,
				}, nil
			}
			// Regular users: require org membership and validate role
			if a.CurrentUserRole() != nil {
				if u, o, err := db.GetUserAccountAndOrg(
					ctx,
					a.CurrentUserID(),
					*a.CurrentOrgID(),
				); errors.Is(err, apierrors.ErrNotFound) {
					return nil, authn.ErrBadAuthentication
				} else if err != nil {
					return nil, err
				} else if u.UserRole != *a.CurrentUserRole() {
					return nil, authn.ErrBadAuthentication
				} else {
					return &DbAuthInfo{
						AuthInfo: &SimpleAuthInfo{
							userID:                 a.CurrentUserID(),
							userEmail:              a.CurrentUserEmail(),
							organizationID:         a.CurrentOrgID(),
							customerOrganizationID: u.CustomerOrganizationID,
							emailVerified:          a.CurrentUserEmailVerified(),
							userRole:               a.CurrentUserRole(),
							isSuperAdmin:           false,
							rawToken:               a.Token(),
						},
						user: util.PtrTo(u.AsUserAccount()),
						org:  o,
					}, nil
				}
			}
			return nil, authn.ErrBadAuthentication
		} else {
			// some special tokens like password reset don't have an organization ID
			if u, err := db.GetUserAccountByID(ctx, a.CurrentUserID()); errors.Is(err, apierrors.ErrNotFound) {
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

type agentDBAuthInfo struct {
	AuthInfo
	org *types.Organization
}

func (a agentDBAuthInfo) CurrentOrg() *types.Organization {
	return a.org
}

func AgentDbAuthenticator() authn.Authenticator[AgentAuthInfo, AuthInfoWithOrganization] {
	fn := func(ctx context.Context, a AgentAuthInfo) (AuthInfoWithOrganization, error) {
		customer, org, err := db.GetCustomerAndOrgForDeploymentTarget(ctx, a.CurrentDeploymentTargetID())
		if errors.Is(err, apierrors.ErrNotFound) {
			return nil, authn.ErrBadAuthentication
		} else if err != nil {
			return nil, err
		}
		info := &SimpleAuthInfo{
			organizationID: &org.ID,
			rawToken:       a.Token(),
		}
		if customer != nil {
			info.customerOrganizationID = &customer.ID
		}
		return &agentDBAuthInfo{AuthInfo: info, org: org}, nil
	}
	return authn.AuthenticatorFunc[AgentAuthInfo, AuthInfoWithOrganization](fn)
}

func DropUser() authn.Authenticator[AuthInfoWithUserAndOrganization, AuthInfoWithOrganization] {
	fn := func(ctx context.Context, a AuthInfoWithUserAndOrganization) (AuthInfoWithOrganization, error) {
		return a, nil
	}
	return authn.AuthenticatorFunc[AuthInfoWithUserAndOrganization, AuthInfoWithOrganization](fn)
}
