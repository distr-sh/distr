package userauth

import (
	"context"
	"errors"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
)

// SetUserPassword hashes the given password, optionally updates the name (when name is non-nil and non-empty),
// and persists the user. The passed user is updated in place with the values returned from the database.
func SetUserPassword(ctx context.Context, user *types.UserAccount, password string, name *string) error {
	if name != nil && *name != "" {
		user.Name = *name
	}
	user.Password = password
	if err := security.HashPassword(user); err != nil {
		return err
	}
	return db.UpdateUserAccount(ctx, user)
}

// VerifyUserEmail marks the given email address (the one carried by the verification token) as verified and
// persists the change. If the token carries a different email address than the user currently has, the email
// is updated as well (used by the email-change flow). It is a no-op when the email is unchanged and already
// verified. The passed user is updated in place with the values returned from the database.
func VerifyUserEmail(ctx context.Context, user *types.UserAccount, email string) error {
	if user.Email == email && user.EmailVerifiedAt != nil {
		return nil
	}
	user.Email = email
	return db.UpdateUserAccountEmailVerified(ctx, user)
}

// PrimaryOrganization returns the organization a login should default to: the user's last used organization,
// or the first one if none was used before. For super admins, who are not assigned to organizations, all
// organizations are considered. It returns apierrors.ErrNotFound when there is no organization to default to.
func PrimaryOrganization(ctx context.Context, user types.UserAccount) (types.OrganizationWithUserRole, error) {
	var orgs []types.OrganizationWithUserRole
	var err error
	if user.IsSuperAdmin {
		orgs, err = db.GetAllOrganizationsForSuperAdmin(ctx)
	} else {
		orgs, err = db.GetOrganizationsForUser(ctx, user.ID)
	}
	if err != nil {
		return types.OrganizationWithUserRole{}, err
	}
	if len(orgs) == 0 {
		return types.OrganizationWithUserRole{}, apierrors.ErrNotFound
	}
	org := orgs[0]
	if user.LastUsedOrganizationID != nil {
		for _, o := range orgs {
			if o.ID == *user.LastUsedOrganizationID {
				org = o
				break
			}
		}
	}
	return org, nil
}

// EnsurePrimaryOrganization returns the organization a login should default to, creating a personal organization
// for the user when they are not part of any organization yet. This mirrors the behavior of a regular login, so
// that a user who was removed from all of their organizations can still complete an invite or password reset.
// A super admin without an organization is an unexpected state and results in an error.
func EnsurePrimaryOrganization(ctx context.Context, user types.UserAccount) (types.OrganizationWithUserRole, error) {
	if org, err := PrimaryOrganization(ctx, user); err == nil {
		return org, nil
	} else if !errors.Is(err, apierrors.ErrNotFound) {
		return types.OrganizationWithUserRole{}, err
	}

	if user.IsSuperAdmin {
		return types.OrganizationWithUserRole{}, errors.New("super admin has no organizations, this should never happen")
	}

	org := types.OrganizationWithUserRole{UserRole: types.UserRoleAdmin}
	org.Name = user.Email
	if err := db.CreateOrganization(ctx, &org.Organization); err != nil {
		return types.OrganizationWithUserRole{}, err
	}
	if err := db.CreateUserAccountOrganizationAssignment(
		ctx, user.ID, org.ID, org.UserRole, org.CustomerOrganizationID, nil); err != nil {
		return types.OrganizationWithUserRole{}, err
	}
	return org, nil
}

// GenerateLoginToken generates a default login token scoped to the user's primary organization, the same kind
// of token that is issued on a regular login. A personal organization is created when the user has none.
func GenerateLoginToken(ctx context.Context, user types.UserAccount) (string, error) {
	org, err := EnsurePrimaryOrganization(ctx, user)
	if err != nil {
		return "", err
	}
	_, token, err := authjwt.GenerateDefaultToken(user, org)
	return token, err
}
