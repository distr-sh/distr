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
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
)

const OIDCStateMaxAge = 10 * time.Minute

type OIDCState struct {
	ID                        uuid.UUID      `db:"id"`
	CreatedAt                 time.Time      `db:"created_at"`
	PKCECodeVerifier          string         `db:"pkce_code_verifier"`
	Nonce                     string         `db:"nonce"`
	CustomOIDCConfigurationID *uuid.UUID     `db:"custom_oidc_configuration_id"`
	Flow                      types.OIDCFlow `db:"flow"`
}

func (s OIDCState) Expired() bool {
	return s.CreatedAt.Before(time.Now().UTC().Add(-OIDCStateMaxAge))
}

func CreateOIDCState(
	ctx context.Context,
	customOIDCConfigurationID *uuid.UUID,
	flow types.OIDCFlow,
) (OIDCState, error) {
	nonce, err := generateOIDCNonce()
	if err != nil {
		return OIDCState{}, err
	}
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO OIDCState AS s (pkce_code_verifier, nonce, custom_oidc_configuration_id, flow)
		VALUES (@pkceCodeVerifier, @nonce, @customOidcConfigurationId, @flow)
		RETURNING s.id, s.created_at, s.pkce_code_verifier, s.nonce, s.custom_oidc_configuration_id, s.flow`,
		pgx.NamedArgs{
			"pkceCodeVerifier":          oauth2.GenerateVerifier(),
			"nonce":                     nonce,
			"customOidcConfigurationId": customOIDCConfigurationID,
			"flow":                      flow,
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

func DeleteOIDCState(ctx context.Context, id uuid.UUID) (OIDCState, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`DELETE FROM OIDCState AS s WHERE s.id = @id
		RETURNING s.id, s.created_at, s.pkce_code_verifier, s.nonce, s.custom_oidc_configuration_id, s.flow`,
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
		`DELETE FROM OIDCState WHERE now() - created_at > @maxAge`,
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
