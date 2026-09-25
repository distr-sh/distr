package authkey

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/authkey"
	"github.com/distr-sh/distr/internal/authn"
)

func Authenticator() authn.Authenticator[string, authkey.Token] {
	return authn.AuthenticatorFunc[string, authkey.Token](
		func(ctx context.Context, encoded string) (authkey.Token, error) {
			if token, err := authkey.Parse(encoded); err != nil {
				return authkey.Token{}, fmt.Errorf("%w: %w", authn.ErrBadAuthentication, err)
			} else {
				return token, nil
			}
		},
	)
}
