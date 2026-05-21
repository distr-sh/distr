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

// FromServiceAccountAuthKey resolves an access token to a service-account-backed AuthInfo.
// The resulting AuthInfo has CurrentServiceAccountID() set, while CurrentUserID() returns uuid.Nil
// and CurrentUserEmail() returns "".
func FromServiceAccountAuthKey(ctx context.Context, token authkey.Key) (AuthInfo, error) {
	at, err := db.GetServiceAccountAccessTokenByKeyUpdatingLastUsed(ctx, token)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			err = fmt.Errorf("%w: %w", authn.ErrBadAuthentication, err)
		}
		return nil, err
	}
	return &SimpleAuthInfo{
		serviceAccountID:       &at.ServiceAccount.ID,
		organizationID:         &at.ServiceAccount.OrganizationID,
		customerOrganizationID: at.ServiceAccount.CustomerOrganizationID,
		accountRole:            &at.ServiceAccount.AccountRole,
		rawToken:               token,
	}, nil
}

// ServiceAccountAuthKeyAuthenticator returns an Authenticator that resolves an access key
// to a service-account-backed AuthInfo.
func ServiceAccountAuthKeyAuthenticator() authn.Authenticator[authkey.Key, AuthInfo] {
	return authn.AuthenticatorFunc[authkey.Key, AuthInfo](FromServiceAccountAuthKey)
}

// UnifiedAuthKeyAuthenticator resolves an access key against the user PAT table first, and
// falls back to the service-account token table if no user-PAT match is found.
func UnifiedAuthKeyAuthenticator() authn.Authenticator[authkey.Key, AuthInfo] {
	return authn.AuthenticatorFunc[authkey.Key, AuthInfo](
		func(ctx context.Context, token authkey.Key) (AuthInfo, error) {
			if info, err := FromAuthKey(ctx, token); err == nil {
				return info, nil
			} else if !errors.Is(err, authn.ErrBadAuthentication) {
				return nil, err
			}
			return FromServiceAccountAuthKey(ctx, token)
		},
	)
}
