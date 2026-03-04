package supportbundle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	extractToken := token.FromQuery("token")
	return authn.AuthenticatorFunc[*http.Request, *types.SupportBundle](
		func(ctx context.Context, r *http.Request) (*types.SupportBundle, error) {
			tokenStr := extractToken(r)
			if tokenStr == "" {
				return nil, authn.ErrNoAuthentication
			}

			tokenBytes, err := hex.DecodeString(tokenStr)
			if err != nil {
				return nil, fmt.Errorf(
					"%w: invalid token encoding", authn.ErrBadAuthentication,
				)
			}

			h := sha256.Sum256(tokenBytes)

			bundleID, err := uuid.Parse(r.PathValue("supportBundleId"))
			if err != nil {
				return nil, fmt.Errorf(
					"%w: invalid bundle ID", authn.ErrBadAuthentication,
				)
			}

			bundle, err := db.GetSupportBundleByCollectToken(
				ctx, bundleID, h[:],
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
