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

func FromAuthKey(ctx context.Context, token authkey.Token) (AuthInfo, error) {
	at, err := db.AuthenticateAccessToken(ctx, token)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			err = fmt.Errorf("%w: %w", authn.ErrBadAuthentication, err)
		}
		return nil, err
	}

	role := at.EffectiveUserRole()
	return &SimpleAuthInfo{
		userID:                 at.UserAccount.ID,
		userEmail:              at.UserAccount.Email,
		emailVerified:          at.UserAccount.EmailVerifiedAt != nil,
		organizationID:         &at.OrganizationID,
		customerOrganizationID: at.CustomerOrganizationID,
		isAccessToken:          true,
		userRole:               &role,
		// Only the key, never the secret: nothing downstream needs to authenticate with the
		// token again, and a credential that is not carried around cannot be leaked.
		rawToken: token.Key,
	}, nil
}

func AuthKeyAuthenticator() authn.Authenticator[authkey.Token, AuthInfo] {
	return authn.AuthenticatorFunc[authkey.Token, AuthInfo](FromAuthKey)
}
