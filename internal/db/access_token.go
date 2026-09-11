package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authkey"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The column order of the secret row expressions must match the field order of
// types.AccessTokenSecret, because an anonymous record is decoded positionally.
const accessTokenOutputExpr = `
	tok.id, tok.created_at, tok.expires_at, tok.last_used_at, tok.label, tok.key,
	tok.user_account_id, tok.organization_id, tok.user_role AS token_user_role,
	CASE WHEN tok.secret_1_hash IS NOT NULL THEN
		(tok.secret_1_salt, tok.secret_1_hash, tok.secret_1_created_at, tok.secret_1_last_used_at)
	END AS secret_1,
	CASE WHEN tok.secret_2_hash IS NOT NULL THEN
		(tok.secret_2_salt, tok.secret_2_hash, tok.secret_2_created_at, tok.secret_2_last_used_at)
	END AS secret_2
`

var accessTokenWithUserAccountOutputExpr = accessTokenOutputExpr + `,
	(` + userAccountOutputExpr + `) AS user_account,
	oua.user_role,
	oua.customer_organization_id
`

func accessTokenSecretPrefix(slot types.AccessTokenSecretSlot) string {
	if slot == types.AccessTokenSecretSlot1 {
		return "secret_1"
	}
	return "secret_2"
}

func CreateAccessToken(ctx context.Context, token *types.AccessToken) error {
	if token.Secret1 == nil {
		return errors.New("could not create access token: no secret given")
	}
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`INSERT INTO AccessToken AS tok (label, expires_at, key, user_account_id, organization_id, user_role,
				secret_1_salt, secret_1_hash, secret_1_created_at)
			VALUES (@label, @expiresAt, @key, @userAccountId, @orgId, @userRole,
				@secretSalt, @secretHash, now())
			RETURNING %v`,
			accessTokenOutputExpr),
		pgx.NamedArgs{
			"label":         token.Label,
			"expiresAt":     token.ExpiresAt,
			"key":           token.Key[:],
			"userAccountId": token.UserAccountID,
			"orgId":         token.OrganizationID,
			"userRole":      token.UserRole,
			"secretSalt":    token.Secret1.Salt,
			"secretHash":    token.Secret1.Hash,
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

type UpdateAccessTokenParams struct {
	UpdateLabel     bool
	Label           *string
	UpdateExpiresAt bool
	ExpiresAt       *time.Time
	UpdateUserRole  bool
	UserRole        *types.UserRole
}

// UpdateAccessToken changes the fields of a token that are not part of the credential itself and
// returns it as it is now, which for a params value that updates nothing is simply the stored row.
func UpdateAccessToken(ctx context.Context, id, userID, orgID uuid.UUID, p UpdateAccessTokenParams) (
	*types.AccessToken, error,
) {
	args := pgx.NamedArgs{"id": id, "userId": userID, "orgId": orgID}
	var setClauses []string
	if p.UpdateLabel {
		setClauses = append(setClauses, "label = nullif(@label, '')")
		args["label"] = p.Label
	}
	if p.UpdateExpiresAt {
		setClauses = append(setClauses, "expires_at = @expiresAt")
		args["expiresAt"] = p.ExpiresAt
	}
	if p.UpdateUserRole {
		setClauses = append(setClauses, "user_role = @userRole")
		args["userRole"] = p.UserRole
	}
	if len(setClauses) == 0 {
		return GetAccessToken(ctx, id, userID, orgID)
	}

	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`UPDATE AccessToken AS tok
			SET %v
			WHERE tok.id = @id AND tok.user_account_id = @userId AND tok.organization_id = @orgId
			RETURNING %v`,
			strings.Join(setClauses, ", "), accessTokenOutputExpr),
		args,
	)
	if err != nil {
		return nil, fmt.Errorf("could not update access token: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AccessToken]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not update access token: %w", err)
	} else {
		return &result, nil
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

func GetAccessToken(ctx context.Context, id, userID, orgID uuid.UUID) (*types.AccessToken, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(`
			SELECT %v
			FROM AccessToken tok
			WHERE tok.id = @id AND tok.user_account_id = @userId AND tok.organization_id = @orgId`,
			accessTokenOutputExpr),
		pgx.NamedArgs{"id": id, "userId": userID, "orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("error querying access token: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AccessToken]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not get token: %w", err)
	} else {
		return &result, nil
	}
}

// GetAccessTokenByKey returns the token identified by the given key. The key is only an
// identifier, so the caller must still verify the secret of the presented token against the
// returned row before treating the request as authenticated.
func GetAccessTokenByKey(ctx context.Context, key authkey.Key) (*types.AccessTokenWithUserAccount, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`SELECT %v
			FROM AccessToken tok
			INNER JOIN UserAccount u ON tok.user_account_id = u.id
			INNER JOIN Organization_UserAccount oua
				ON oua.user_account_id = tok.user_account_id AND oua.organization_id = tok.organization_id
			WHERE tok.key = @key AND (tok.expires_at IS NULL OR tok.expires_at > now())
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

// MarkAccessTokenUsed records the current time on the token and, unless the token authenticated
// without a secret, on the secret that was used. Because the secret was verified against a row
// that was read earlier, it repeats that verification and returns apierrors.ErrNotFound when the
// token no longer matches, so that a credential revoked in between does not authenticate.
func MarkAccessTokenUsed(
	ctx context.Context,
	id uuid.UUID,
	slot *types.AccessTokenSecretSlot,
	hash []byte,
) error {
	db := internalctx.GetDb(ctx)
	args := pgx.NamedArgs{"id": id}
	secretExpr := ""
	secretCondition := "tok.secret_1_hash IS NULL AND tok.secret_2_hash IS NULL"
	if slot != nil {
		prefix := accessTokenSecretPrefix(*slot)
		secretExpr = fmt.Sprintf(", %v_last_used_at = now()", prefix)
		secretCondition = fmt.Sprintf("tok.%v_hash = @hash", prefix)
		args["hash"] = hash
	}
	cmd, err := db.Exec(
		ctx,
		fmt.Sprintf(
			`UPDATE AccessToken AS tok
			SET last_used_at = now()%v
			WHERE tok.id = @id AND %v AND (tok.expires_at IS NULL OR tok.expires_at > now())`,
			secretExpr, secretCondition),
		args,
	)
	if err != nil {
		return fmt.Errorf("could not update access token: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return fmt.Errorf("could not update access token: %w", apierrors.ErrNotFound)
	}
	return nil
}

// CreateAccessTokenSecret fills the given slot and returns apierrors.ErrConflict if it is
// already occupied, so that a concurrent request can never overwrite a secret that is in use.
func CreateAccessTokenSecret(
	ctx context.Context,
	id, userID, orgID uuid.UUID,
	slot types.AccessTokenSecretSlot,
	secret types.AccessTokenSecret,
) (*types.AccessToken, error) {
	db := internalctx.GetDb(ctx)
	prefix := accessTokenSecretPrefix(slot)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`UPDATE AccessToken AS tok
			SET %[1]v_salt = @salt, %[1]v_hash = @hash, %[1]v_created_at = now(), %[1]v_last_used_at = NULL
			WHERE tok.id = @id AND tok.user_account_id = @userId AND tok.organization_id = @orgId
				AND tok.%[1]v_hash IS NULL
			RETURNING %[2]v`,
			prefix, accessTokenOutputExpr,
		),
		pgx.NamedArgs{
			"id":     id,
			"userId": userID,
			"orgId":  orgID,
			"salt":   secret.Salt,
			"hash":   secret.Hash,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not create access token secret: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.AccessToken]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrConflict
		}
		return nil, fmt.Errorf("could not create access token secret: %w", err)
	} else {
		return &result, nil
	}
}

// DeleteAccessTokenSecret clears the given slot. It requires the other slot to be filled,
// because a token whose last secret was removed would fall back to authenticating on its key
// alone. Deleting the last secret returns apierrors.ErrConflict.
func DeleteAccessTokenSecret(
	ctx context.Context,
	id, userID, orgID uuid.UUID,
	slot types.AccessTokenSecretSlot,
) error {
	db := internalctx.GetDb(ctx)
	prefix := accessTokenSecretPrefix(slot)
	cmd, err := db.Exec(
		ctx,
		fmt.Sprintf(
			`UPDATE AccessToken AS tok
			SET %[1]v_salt = NULL, %[1]v_hash = NULL, %[1]v_created_at = NULL, %[1]v_last_used_at = NULL
			WHERE tok.id = @id AND tok.user_account_id = @userId AND tok.organization_id = @orgId
				AND tok.%[1]v_hash IS NOT NULL AND tok.%[2]v_hash IS NOT NULL`,
			prefix, accessTokenSecretPrefix(slot.Other()),
		),
		pgx.NamedArgs{"id": id, "userId": userID, "orgId": orgID},
	)
	if err != nil {
		return fmt.Errorf("could not delete access token secret: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrConflict
	}
	return nil
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
