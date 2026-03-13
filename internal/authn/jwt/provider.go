package jwt

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/authn"
	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func Authenticator(jwtAuthGetter func() *jwtauth.JWTAuth) authn.Authenticator[string, jwt.Token] {
	return authn.AuthenticatorFunc[string, jwt.Token](
		func(ctx context.Context, s string) (jwt.Token, error) {
			if token, err := jwtauth.VerifyToken(jwtAuthGetter(), s); err != nil {
				return nil, fmt.Errorf("%w: %w", authn.ErrBadAuthentication, err)
			} else {
				return token, nil
			}
		},
	)
}
