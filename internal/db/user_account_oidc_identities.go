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

// GetUserAccountWithOIDCIdentity returns the user account linked to the given identity provider
// identity, together with the identity itself. customOIDCConfigurationID selects the organization
// configuration the identity belongs to and is nil for the instance-scoped providers: the same
// (issuer, subject) may be known through an instance provider and through any number of
// organization configurations, and those identities must never be confused for one another.
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

// GetUserAccountOIDCIdentities returns the identities of the given account. Custom identities
// carry the name of the configuration and of the organization controlling it, so that an
// organization-scoped login is never presented as an instance-wide one.
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

// ExistsUserAccountCustomOIDCIdentity reports whether the account can sign in through an
// organization's own identity provider. Such an account must be a member of that one organization
// and of no other, so this gates every operation that would add a membership.
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
		VALUES (@userId, @provider, @issuer, @subject, @email, current_timestamp, @customOidcConfigurationId)
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
		SET last_login_at = current_timestamp, email = @email
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
