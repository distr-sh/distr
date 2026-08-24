package jwt

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/authn"
	"github.com/lestrrat-go/jwx/v4/jwt"
)

func Authenticator(verify func(string) (jwt.Token, error)) authn.Authenticator[string, jwt.Token] {
	return authn.AuthenticatorFunc[string, jwt.Token](
		func(ctx context.Context, s string) (jwt.Token, error) {
			if token, err := verify(s); err != nil {
				return nil, fmt.Errorf("%w: %w", authn.ErrBadAuthentication, err)
			} else {
				return token, nil
			}
		},
	)
}
