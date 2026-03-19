package supportbundle

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/authn/token"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func Authenticator() authn.RequestAuthenticator[*types.SupportBundle] {
	extractSecret := token.FromQuery("bundleSecret")
	return authn.AuthenticatorFunc[*http.Request, *types.SupportBundle](
		func(ctx context.Context, r *http.Request) (*types.SupportBundle, error) {
			bundleSecret := extractSecret(r)
			if bundleSecret == "" {
				return nil, authn.ErrNoAuthentication
			}

			bundleID, err := uuid.Parse(r.PathValue("bundleId"))
			if err != nil {
				return nil, fmt.Errorf(
					"%w: invalid bundle ID", authn.ErrBadAuthentication,
				)
			}

			bundle, err := db.GetSupportBundleByBundleSecret(
				ctx, bundleID, bundleSecret,
			)
			if errors.Is(err, apierrors.ErrNotFound) {
				return nil, fmt.Errorf(
					"%w: invalid or expired token",
					authn.ErrBadAuthentication,
				)
			}
			if err != nil {
				return nil, err
			}

			return bundle, nil
		},
	)
}
