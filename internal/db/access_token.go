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
	accessTokenOutputExpr = `
	tok.id, tok.created_at, tok.expires_at, tok.last_used_at, tok.label, tok.key,
	tok.user_account_id, tok.organization_id, tok.user_role AS token_user_role
`
	accessTokenWithUserAccountOutputExpr = accessTokenOutputExpr + `,
	(` + userAccountOutputExpr + `) AS user_account,
	oua.user_role,
	oua.customer_organization_id
`
)

func CreateAccessToken(ctx context.Context, token *types.AccessToken) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`INSERT INTO AccessToken AS tok (label, expires_at, key, user_account_id, organization_id, user_role)
			VALUES (@label, @expiresAt, @key, @userAccountId, @orgId, @userRole)
			RETURNING %v`,
			accessTokenOutputExpr),
		pgx.NamedArgs{
			"label":         token.Label,
			"expiresAt":     token.ExpiresAt,
			"key":           token.Key[:],
			"userAccountId": token.UserAccountID,
			"orgId":         token.OrganizationID,
			"userRole":      token.UserRole,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create access token: %w", err)
	}
	if res, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AccessToken]); err != nil {
		return fmt.Errorf("could not create access token: %w", err)
	} else {
		*token = res
		return nil
	}
}

func DeleteAccessToken(ctx context.Context, id, userID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	if _, err := db.Exec(
		ctx,
		"DELETE FROM AccessToken WHERE id = @id AND user_account_id = @userId",
		pgx.NamedArgs{"id": id, "userId": userID},
	); err != nil {
		return fmt.Errorf("could not delete token: %w", err)
	}
	return nil
}

func GetAccessTokens(ctx context.Context, userID, orgID uuid.UUID) ([]types.AccessToken, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(`
			SELECT %v
			FROM AccessToken tok
			WHERE tok.user_account_id = @userId AND tok.organization_id = @orgId`, accessTokenOutputExpr),
		pgx.NamedArgs{"userId": userID, "orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("error querying access tokens: %w", err)
	}
	if result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AccessToken]); err != nil {
		return nil, fmt.Errorf("could not get tokens: %w", err)
	} else {
		return result, nil
	}
}

func GetAccessTokenByKeyUpdatingLastUsed(
	ctx context.Context,
	key authkey.Key,
) (*types.AccessTokenWithUserAccount, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`WITH updated AS (
				UPDATE AccessToken
				SET last_used_at = now()
				WHERE key = @key AND (expires_at IS NULL OR expires_at > now())
				RETURNING *
			)
			SELECT %v FROM updated tok
			INNER JOIN UserAccount u ON tok.user_account_id = u.id
			LEFT JOIN Organization_UserAccount oua
				ON oua.user_account_id = tok.user_account_id AND oua.organization_id = tok.organization_id
			`,
			accessTokenWithUserAccountOutputExpr,
		),
		pgx.NamedArgs{"key": key[:]},
	)
	if err != nil {
		return nil, fmt.Errorf("error querying access token: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AccessTokenWithUserAccount]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not get token: %w", err)
	} else {
		return &result, nil
	}
}

func DeleteAccessTokensOfUserInOrg(ctx context.Context, userID, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	if _, err := db.Exec(
		ctx,
		"DELETE FROM AccessToken WHERE user_account_id = @userId AND organization_id = @orgId",
		pgx.NamedArgs{"userId": userID, "orgId": orgID},
	); err != nil {
		return fmt.Errorf("could not delete tokens: %w", err)
	}
	return nil
}
