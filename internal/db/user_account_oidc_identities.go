package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const userAccountOIDCIdentityOutputExpr = `
	i.id,
	i.created_at,
	i.user_account_id,
	i.provider,
	i.issuer,
	i.subject,
	i.email,
	i.last_login_at,
	i.custom_oidc_configuration_id`

func GetUserAccountWithOIDCIdentity(
	ctx context.Context,
	customOIDCConfigurationID *uuid.UUID,
	issuer, subject string,
) (
	*types.UserAccount,
	*types.UserAccountOIDCIdentity,
	error,
) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT ("+userAccountOutputExpr+`),
			(`+userAccountOIDCIdentityOutputExpr+`)
		FROM UserAccountOIDCIdentity i
		INNER JOIN UserAccount u ON u.id = i.user_account_id
		WHERE i.issuer = @issuer AND i.subject = @subject
			AND i.custom_oidc_configuration_id IS NOT DISTINCT FROM @customOidcConfigurationId`,
		pgx.NamedArgs{
			"issuer":                    issuer,
			"subject":                   subject,
			"customOidcConfigurationId": customOIDCConfigurationID,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not query UserAccountOIDCIdentity: %w", err)
	}
	res, err := pgx.CollectExactlyOneRow[struct {
		User     types.UserAccount
		Identity types.UserAccountOIDCIdentity
	}](rows, pgx.RowToStructByPos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, apierrors.ErrNotFound
		}
		return nil, nil, fmt.Errorf("could not map UserAccountOIDCIdentity: %w", err)
	}
	return &res.User, &res.Identity, nil
}

func GetUserAccountOIDCIdentities(ctx context.Context, userID uuid.UUID) (
	[]types.UserAccountOIDCIdentityWithConfiguration,
	error,
) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+userAccountOIDCIdentityOutputExpr+`, c.name, o.name
		FROM UserAccountOIDCIdentity i
		LEFT JOIN CustomOIDCConfiguration c ON c.id = i.custom_oidc_configuration_id
		LEFT JOIN Organization o ON o.id = c.organization_id
		WHERE i.user_account_id = @userId
		ORDER BY i.created_at`,
		pgx.NamedArgs{"userId": userID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query UserAccountOIDCIdentity: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByPos[types.UserAccountOIDCIdentityWithConfiguration])
	if err != nil {
		return nil, fmt.Errorf("could not map UserAccountOIDCIdentity: %w", err)
	}
	return result, nil
}

func ExistsUserAccountCustomOIDCIdentity(ctx context.Context, userID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM UserAccountOIDCIdentity
			WHERE user_account_id = @userId AND custom_oidc_configuration_id IS NOT NULL
		)`,
		pgx.NamedArgs{"userId": userID},
	)
	if err != nil {
		return false, fmt.Errorf("could not query UserAccountOIDCIdentity: %w", err)
	}
	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, fmt.Errorf("could not query UserAccountOIDCIdentity: %w", err)
	}
	return exists, nil
}

func ExistsUserAccountCustomOIDCIdentityExcept(ctx context.Context, userID, organizationID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM UserAccountOIDCIdentity i
			INNER JOIN CustomOIDCConfiguration c ON c.id = i.custom_oidc_configuration_id
			WHERE i.user_account_id = @userId AND c.organization_id != @organizationId
		)`,
		pgx.NamedArgs{"userId": userID, "organizationId": organizationID},
	)
	if err != nil {
		return false, fmt.Errorf("could not query UserAccountOIDCIdentity: %w", err)
	}
	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, fmt.Errorf("could not query UserAccountOIDCIdentity: %w", err)
	}
	return exists, nil
}

func CountUserAccountOIDCIdentities(ctx context.Context, userID uuid.UUID) (int64, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT count(*) FROM UserAccountOIDCIdentity WHERE user_account_id = @userId`,
		pgx.NamedArgs{"userId": userID},
	)
	if err != nil {
		return 0, fmt.Errorf("could not count UserAccountOIDCIdentity: %w", err)
	}
	count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, fmt.Errorf("could not count UserAccountOIDCIdentity: %w", err)
	}
	return count, nil
}

func CreateUserAccountOIDCIdentity(ctx context.Context, identity *types.UserAccountOIDCIdentity) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO UserAccountOIDCIdentity AS i
			(user_account_id, provider, issuer, subject, email, last_login_at, custom_oidc_configuration_id)
		VALUES (@userId, @provider, @issuer, @subject, @email, now(), @customOidcConfigurationId)
		RETURNING `+userAccountOIDCIdentityOutputExpr,
		pgx.NamedArgs{
			"userId":                    identity.UserAccountID,
			"provider":                  identity.Provider,
			"issuer":                    identity.Issuer,
			"subject":                   identity.Subject,
			"email":                     identity.Email,
			"customOidcConfigurationId": identity.CustomOIDCConfigurationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create UserAccountOIDCIdentity: %w", err)
	}
	created, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[types.UserAccountOIDCIdentity])
	if err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("identity %v/%v is already linked: %w",
				identity.Issuer, identity.Subject, apierrors.ErrAlreadyExists)
		}
		return fmt.Errorf("could not create UserAccountOIDCIdentity: %w", err)
	}
	*identity = created
	return nil
}

// UpdateUserAccountOIDCIdentityOnLogin records the login and the email the identity provider
// reported for it. The email of the user account itself is deliberately left untouched.
func UpdateUserAccountOIDCIdentityOnLogin(ctx context.Context, id uuid.UUID, email *string) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`UPDATE UserAccountOIDCIdentity
		SET last_login_at = now(), email = @email
		WHERE id = @id`,
		pgx.NamedArgs{"id": id, "email": email},
	)
	if err == nil && cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("could not update UserAccountOIDCIdentity: %w", err)
	}
	return nil
}

func DeleteCustomOIDCIdentitiesOfUserInOrg(ctx context.Context, userID, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	if _, err := db.Exec(ctx,
		`DELETE FROM UserAccountOIDCIdentity
		WHERE user_account_id = @userId AND custom_oidc_configuration_id IN (
			SELECT id FROM CustomOIDCConfiguration WHERE organization_id = @orgId
		)`,
		pgx.NamedArgs{"userId": userID, "orgId": orgID},
	); err != nil {
		return fmt.Errorf("could not delete UserAccountOIDCIdentity: %w", err)
	}
	return nil
}

func DeleteUserAccountOIDCIdentity(ctx context.Context, userID, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`DELETE FROM UserAccountOIDCIdentity WHERE id = @id AND user_account_id = @userId`,
		pgx.NamedArgs{"id": id, "userId": userID},
	)
	if err == nil && cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("could not delete UserAccountOIDCIdentity: %w", err)
	}
	return nil
}
