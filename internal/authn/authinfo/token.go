package authinfo

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authkey"
	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/db"
)

func FromAuthKey(ctx context.Context, token authkey.Key) (AuthInfo, error) {
	if at, err := db.GetAccessTokenByKeyUpdatingLastUsed(ctx, token); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			err = fmt.Errorf("%w: %w", authn.ErrBadAuthentication, err)
		}
		return nil, err
	} else {
		// A token issued by a super admin carries the super admin identity, so it
		// has the same read access as an interactive super admin session (it can
		// read all organizations). Write access is still blocked for super admins
		// (BlockSuperAdmin), and EffectiveUserRole caps the role to read-only.
		// Such tokens can only be created in an organization the super admin is a
		// member of (enforced at creation).
		role := at.EffectiveUserRole()
		return &SimpleAuthInfo{
			userID:                 at.UserAccount.ID,
			userEmail:              at.UserAccount.Email,
			emailVerified:          at.UserAccount.EmailVerifiedAt != nil,
			organizationID:         &at.OrganizationID,
			customerOrganizationID: at.CustomerOrganizationID,
			userRole:               &role,
			isSuperAdmin:           at.UserAccount.IsSuperAdmin,
			rawToken:               token,
		}, nil
	}
}

func AuthKeyAuthenticator() authn.Authenticator[authkey.Key, AuthInfo] {
	return authn.AuthenticatorFunc[authkey.Key, AuthInfo](FromAuthKey)
}
