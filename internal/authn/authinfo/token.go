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
	at, err := db.GetAccessTokenByKey(ctx, token.Key)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			err = fmt.Errorf("%w: %w", authn.ErrBadAuthentication, err)
		}
		return nil, err
	}

	slot, err := at.VerifySecret(token.Secret)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", authn.ErrBadAuthentication, err)
	} else if err := db.MarkAccessTokenUsed(ctx, at.ID, slot); err != nil {
		return nil, err
	}

	role := at.EffectiveUserRole()
	return &SimpleAuthInfo{
		userID:                 at.UserAccount.ID,
		userEmail:              at.UserAccount.Email,
		emailVerified:          at.UserAccount.EmailVerifiedAt != nil,
		organizationID:         &at.OrganizationID,
		customerOrganizationID: at.CustomerOrganizationID,
		// An access token is created for one organization and is not proof that its owner is
		// present, so it must not reach beyond the session it was created from.
		organizationScoped: true,
		userRole:           &role,
		// Only the key, never the secret: nothing downstream needs to authenticate with the
		// token again, and a credential that is not carried around cannot be leaked.
		rawToken: token.Key,
	}, nil
}

func AuthKeyAuthenticator() authn.Authenticator[authkey.Token, AuthInfo] {
	return authn.AuthenticatorFunc[authkey.Token, AuthInfo](FromAuthKey)
}
