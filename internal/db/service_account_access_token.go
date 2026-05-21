package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authkey"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	serviceAccountAccessTokenOutputExpr = `
	tok.id, tok.created_at, tok.expires_at, tok.last_used_at, tok.label, tok.key, tok.service_account_id
`
	serviceAccountAccessTokenWithServiceAccountOutputExpr = serviceAccountAccessTokenOutputExpr + `,
	(` + serviceAccountOutputExpr + `) AS service_account
`
)

func CreateServiceAccountAccessToken(ctx context.Context, token *types.ServiceAccountAccessToken) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`INSERT INTO ServiceAccountAccessToken AS tok (label, expires_at, key, service_account_id)
			VALUES (@label, @expiresAt, @key, @serviceAccountId)
			RETURNING %s`,
			serviceAccountAccessTokenOutputExpr,
		),
		pgx.NamedArgs{
			"label":            token.Label,
			"expiresAt":        token.ExpiresAt,
			"key":              token.Key[:],
			"serviceAccountId": token.ServiceAccountID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create service account token: %w", err)
	}
	if res, err := pgx.CollectExactlyOneRow(
		rows,
		pgx.RowToStructByName[types.ServiceAccountAccessToken],
	); err != nil {
		return fmt.Errorf("could not create service account token: %w", err)
	} else {
		*token = res
		return nil
	}
}

func DeleteServiceAccountAccessToken(ctx context.Context, id, serviceAccountID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM ServiceAccountAccessToken WHERE id = @id AND service_account_id = @serviceAccountId`,
		pgx.NamedArgs{"id": id, "serviceAccountId": serviceAccountID},
	)
	if err != nil {
		return fmt.Errorf("could not delete service account token: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}

func GetServiceAccountAccessTokens(
	ctx context.Context,
	serviceAccountID uuid.UUID,
) ([]types.ServiceAccountAccessToken, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`SELECT %s
			FROM ServiceAccountAccessToken tok
			WHERE tok.service_account_id = @serviceAccountId
			ORDER BY tok.created_at DESC`,
			serviceAccountAccessTokenOutputExpr,
		),
		pgx.NamedArgs{"serviceAccountId": serviceAccountID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query service account tokens: %w", err)
	}
	if result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ServiceAccountAccessToken]); err != nil {
		return nil, fmt.Errorf("could not map service account tokens: %w", err)
	} else {
		return result, nil
	}
}

func GetServiceAccountAccessTokenByKeyUpdatingLastUsed(
	ctx context.Context,
	key authkey.Key,
) (*types.ServiceAccountAccessTokenWithServiceAccount, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`WITH updated AS (
				UPDATE ServiceAccountAccessToken
				SET last_used_at = now()
				WHERE key = @key AND (expires_at IS NULL OR expires_at > now())
				RETURNING *
			)
			SELECT %s FROM updated tok
			INNER JOIN ServiceAccount sa ON sa.id = tok.service_account_id
			`,
			serviceAccountAccessTokenWithServiceAccountOutputExpr,
		),
		pgx.NamedArgs{"key": key[:]},
	)
	if err != nil {
		return nil, fmt.Errorf("error querying service account token: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(
		rows,
		pgx.RowToStructByName[types.ServiceAccountAccessTokenWithServiceAccount],
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not get service account token: %w", err)
	} else {
		return &result, nil
	}
}
