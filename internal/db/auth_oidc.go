package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

// OIDCStateMaxAge is how long an in-flight authorization request stays redeemable. It has to
// cover a full interactive login at the identity provider, including a password prompt, a
// multi-factor challenge and a consent screen.
const OIDCStateMaxAge = 10 * time.Minute

// OIDCState is an in-flight authorization request. It binds the callback to the flow that
// started it: the PKCE verifier and the nonce are only known to this server, and the
// configuration id prevents a code issued for one provider from being redeemed against another.
type OIDCState struct {
	ID                        uuid.UUID  `db:"id"`
	CreatedAt                 time.Time  `db:"created_at"`
	PKCECodeVerifier          string     `db:"pkce_code_verifier"`
	Nonce                     string     `db:"nonce"`
	CustomOIDCConfigurationID *uuid.UUID `db:"custom_oidc_configuration_id"`
}

// Expired reports whether the state is too old to be redeemed.
func (s OIDCState) Expired() bool {
	return s.CreatedAt.Before(time.Now().UTC().Add(-OIDCStateMaxAge))
}

// CreateOIDCState creates the state for a new authorization request. customOIDCConfigurationID is
// nil for the instance-scoped providers.
func CreateOIDCState(ctx context.Context, customOIDCConfigurationID *uuid.UUID) (OIDCState, error) {
	nonce, err := generateOIDCNonce()
	if err != nil {
		return OIDCState{}, err
	}
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO OIDCState AS s (pkce_code_verifier, nonce, custom_oidc_configuration_id)
		VALUES (@pkceCodeVerifier, @nonce, @customOidcConfigurationId)
		RETURNING s.id, s.created_at, s.pkce_code_verifier, s.nonce, s.custom_oidc_configuration_id`,
		pgx.NamedArgs{
			"pkceCodeVerifier":          oauth2.GenerateVerifier(),
			"nonce":                     nonce,
			"customOidcConfigurationId": customOIDCConfigurationID,
		})
	if err != nil {
		return OIDCState{}, fmt.Errorf("could not insert OIDCState: %w", err)
	}
	state, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[OIDCState])
	if err != nil {
		return OIDCState{}, fmt.Errorf("could not insert OIDCState: %w", err)
	}
	return state, nil
}

// DeleteOIDCState consumes the state, so an authorization code can only be redeemed once.
func DeleteOIDCState(ctx context.Context, id uuid.UUID) (OIDCState, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`DELETE FROM OIDCState AS s WHERE s.id = @id
		RETURNING s.id, s.created_at, s.pkce_code_verifier, s.nonce, s.custom_oidc_configuration_id`,
		pgx.NamedArgs{"id": id})
	if err != nil {
		return OIDCState{}, fmt.Errorf("could not delete OIDCState: %w", err)
	}
	state, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[OIDCState])
	if errors.Is(err, pgx.ErrNoRows) {
		return OIDCState{}, apierrors.ErrNotFound
	} else if err != nil {
		return OIDCState{}, fmt.Errorf("could not delete OIDCState: %w", err)
	}
	return state, nil
}

func CleanupOIDCStates(ctx context.Context) (int64, error) {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM OIDCState WHERE current_timestamp - created_at > @maxAge`,
		pgx.NamedArgs{"maxAge": OIDCStateMaxAge},
	)
	if err != nil {
		return 0, fmt.Errorf("error cleaning up OIDCState: %w", err)
	} else {
		return cmd.RowsAffected(), nil
	}
}

func generateOIDCNonce() (string, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("could not generate OIDC nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(nonce), nil
}
