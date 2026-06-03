package auth

import (
	"context"

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
// or the first one if none was used before. It returns apierrors.ErrNotFound when the user is not part of any
// organization.
func PrimaryOrganization(ctx context.Context, user types.UserAccount) (types.OrganizationWithUserRole, error) {
	orgs, err := db.GetOrganizationsForUser(ctx, user.ID)
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

// GenerateLoginToken generates a default login token scoped to the user's primary organization, the same kind
// of token that is issued on a regular login.
func GenerateLoginToken(ctx context.Context, user types.UserAccount) (string, error) {
	org, err := PrimaryOrganization(ctx, user)
	if err != nil {
		return "", err
	}
	_, token, err := authjwt.GenerateDefaultToken(user, org)
	return token, err
}
