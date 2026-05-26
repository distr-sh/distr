package authinfo

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authkey"
	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/util"
)

func FromAuthKey(ctx context.Context, token authkey.Key) (AuthInfo, error) {
	if at, err := db.GetAccessTokenByKeyUpdatingLastUsed(ctx, token); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			err = fmt.Errorf("%w: %w", authn.ErrBadAuthentication, err)
		}
		return nil, err
	} else {
		return &SimpleAuthInfo{
			userID:                 at.UserAccount.ID,
			userEmail:              at.UserAccount.Email,
			emailVerified:          at.UserAccount.EmailVerifiedAt != nil,
			organizationID:         &at.OrganizationID,
			customerOrganizationID: at.CustomerOrganizationID,
			userRole:               util.PtrTo(at.EffectiveUserRole()),
			rawToken:               token,
		}, nil
	}
}

func AuthKeyAuthenticator() authn.Authenticator[authkey.Key, AuthInfo] {
	return authn.AuthenticatorFunc[authkey.Key, AuthInfo](FromAuthKey)
}
