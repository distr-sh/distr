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

const (
	userAccountOutputExpr = `
		u.id,
		u.created_at,
		u.email,
		u.email_verified_at,
		u.email_verified_at IS NOT NULL,
		u.password_hash,
		u.password_salt,
		u.name,
		u.image_id,
		u.last_used_organization_id,
		u.mfa_secret,
		u.mfa_enabled,
		u.mfa_enabled_at`
	userAccountWithRoleOutputExpr = userAccountOutputExpr +
		", j.user_role, j.created_at, j.customer_organization_id "
	userAccountWithRoleOutputExprWithAlias = userAccountWithRoleOutputExpr + " as joined_org_at "
)

func CreateUserAccountWithOrganization(
	ctx context.Context,
	userAccount *types.UserAccount,
) (*types.Organization, error) {
	org := types.Organization{
		Name: userAccount.Email,
	}
	if err := CreateUserAccount(ctx, userAccount); err != nil {
		return nil, err
	} else if err := CreateOrganization(ctx, &org); err != nil {
		return nil, err
	} else if err := CreateUserAccountOrganizationAssignment(
		ctx,
		userAccount.ID,
		org.ID,
		types.UserRoleAdmin,
		nil,
	); err != nil {
		return nil, err
	} else {
		return &org, nil
	}
}

func CreateUserAccount(ctx context.Context, userAccount *types.UserAccount) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"INSERT INTO UserAccount AS u (email, password_hash, password_salt, name, email_verified_at) "+
			"VALUES (@email, @password_hash, @password_salt, @name, @email_verified_at) "+
			"RETURNING "+userAccountOutputExpr,
		pgx.NamedArgs{
			"email":             userAccount.Email,
			"password_hash":     userAccount.PasswordHash,
			"password_salt":     userAccount.PasswordSalt,
			"name":              userAccount.Name,
			"email_verified_at": userAccount.EmailVerifiedAt,
		},
	)
	if err != nil {
		return fmt.Errorf("could not query users: %w", err)
	} else if created, err := pgx.CollectExactlyOneRow[types.UserAccount](rows, pgx.RowToStructByPos); err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("user account with email %v can not be created: %w", userAccount.Email, apierrors.ErrAlreadyExists)
		}
		return fmt.Errorf("could not create user: %w", err)
	} else {
		*userAccount = created
		return nil
	}
}

func UpdateUserAccount(ctx context.Context, userAccount *types.UserAccount) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE UserAccount AS u
		SET email = @email,
			name = @name,
			password_hash = @password_hash,
			password_salt = @password_salt,
			email_verified_at = @email_verified_at,
			image_id = @image_id
		WHERE id = @id
		RETURNING `+userAccountOutputExpr,
		pgx.NamedArgs{
			"id":                userAccount.ID,
			"email":             userAccount.Email,
			"password_hash":     userAccount.PasswordHash,
			"password_salt":     userAccount.PasswordSalt,
			"name":              userAccount.Name,
			"email_verified_at": userAccount.EmailVerifiedAt,
			"image_id":          userAccount.ImageID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not query users: %w", err)
	} else if created, err := pgx.CollectExactlyOneRow[types.UserAccount](rows, pgx.RowToStructByPos); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.ErrNotFound
		} else if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("can not update user with email %v: %w", userAccount.Email, apierrors.ErrAlreadyExists)
		}
		return fmt.Errorf("could not update user: %w", err)
	} else {
		*userAccount = created
		return nil
	}
}

func UpdateUserAccountEmailVerified(ctx context.Context, userAccount *types.UserAccount) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE UserAccount AS u
		SET email = @email, email_verified_at = now()
		WHERE id = @id
		RETURNING `+userAccountOutputExpr,
		pgx.NamedArgs{
			"id":    userAccount.ID,
			"email": userAccount.Email,
		},
	)
	if err != nil {
		return fmt.Errorf("could not query users: %w", err)
	} else if created, err := pgx.CollectExactlyOneRow[types.UserAccount](rows, pgx.RowToStructByPos); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.ErrNotFound
		} else if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("can not update user with email %v: %w", userAccount.Email, apierrors.ErrAlreadyExists)
		}
		return fmt.Errorf("could not update user: %w", err)
	} else {
		*userAccount = created
		return nil
	}
}

func UpdateUserAccountLastUsedOrganizationID(ctx context.Context, userID, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`UPDATE UserAccount SET last_used_organization_id = @orgId WHERE id = @userId`,
		pgx.NamedArgs{"orgId": orgID, "userId": userID},
	)
	if err != nil {
	} else if cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("could not update UserAccount: %w", err)
	}

	return nil
}

func DeleteUserAccountWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `DELETE FROM UserAccount WHERE id = @id`, pgx.NamedArgs{"id": id})
	if err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
	} else if cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("could not delete UserAccount: %w", err)
	}

	return nil
}

func DeleteUserAccountFromOrganization(ctx context.Context, userID, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `
		DELETE FROM Organization_UserAccount
		WHERE user_account_id = @userId AND organization_id = @orgId`,
		pgx.NamedArgs{"userId": userID, "orgId": orgID})
	if err == nil && cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}
	return err
}

func CreateUserAccountOrganizationAssignment(
	ctx context.Context,
	userID, orgID uuid.UUID,
	role types.UserRole,
	customerOrganizationID *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(ctx,
		"INSERT INTO Organization_UserAccount (organization_id, user_account_id, user_role, customer_organization_id) "+
			"VALUES (@orgId, @userId, @role, @customerOrganizationID)",
		pgx.NamedArgs{
			"userId":                 userID,
			"orgId":                  orgID,
			"role":                   role,
			"customerOrganizationID": customerOrganizationID,
		},
	)
	if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
		return apierrors.ErrAlreadyExists
	}
	return err
}

func UpdateUserAccountOrganizationAssignment(
	ctx context.Context,
	userID, orgID uuid.UUID,
	role types.UserRole,
	customerOrganizationID *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		"UPDATE Organization_UserAccount SET user_role = @role, customer_organization_id = @customerOrganizationID "+
			"WHERE organization_id = @orgId AND user_account_id = @userId",
		pgx.NamedArgs{
			"userId":                 userID,
			"orgId":                  orgID,
			"role":                   role,
			"customerOrganizationID": customerOrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update Organization_UserAccount: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return fmt.Errorf("%w: user not found in org", apierrors.ErrNotFound)
	} else {
		return nil
	}
}

func UpdateAllUserAccountOrganizationAssignmentsWithOrganizationID(
	ctx context.Context,
	orgID uuid.UUID,
	role types.UserRole,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(ctx,
		`UPDATE Organization_UserAccount
		SET user_role = @role
		WHERE organization_id = @orgId`,
		pgx.NamedArgs{
			"orgId": orgID,
			"role":  role,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update Organization_UserAccount: %w", err)
	} else {
		return nil
	}
}

func UpdateAllUserAccountOrganizationAssignmentsWithOrganizationSuscriptionType(
	ctx context.Context,
	subscriptionType []types.SubscriptionType,
	role types.UserRole,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(ctx,
		`UPDATE Organization_UserAccount
		SET user_role = @role
		WHERE organization_id IN (
			SELECT id FROM Organization WHERE subscription_type = ANY(@subscriptionType)
		)`,
		pgx.NamedArgs{
			"subscriptionType": subscriptionType,
			"role":             role,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update Organization_UserAccount: %w", err)
	} else {
		return nil
	}
}

func GetUserAccountsByOrgID(ctx context.Context, orgID uuid.UUID) (
	[]types.UserAccountWithUserRole,
	error,
) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+userAccountWithRoleOutputExprWithAlias+`
		FROM UserAccount u
		INNER JOIN Organization_UserAccount j ON u.id = j.user_account_id
		WHERE j.organization_id = @orgId
		ORDER BY u.name, u.email`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query users: %w", err)
	} else if result, err := pgx.CollectRows[types.UserAccountWithUserRole](rows, pgx.RowToStructByPos); err != nil {
		return nil, fmt.Errorf("could not map users: %w", err)
	} else {
		return result, nil
	}
}

func CountVendorUserAccountsByOrgID(ctx context.Context, orgID uuid.UUID) (int64, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT count(*)
		FROM Organization_UserAccount
		WHERE organization_id = @orgId
		  	AND customer_organization_id IS NULL`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	}

	if count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int64]); err != nil {
		return 0, fmt.Errorf("failed to get user count: %w", err)
	} else {
		return count, nil
	}
}

func GetUserAccountsByCustomerOrgID(ctx context.Context, customerOrganizationID uuid.UUID) (
	[]types.UserAccountWithUserRole,
	error,
) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+userAccountWithRoleOutputExprWithAlias+`
		FROM UserAccount u
		INNER JOIN Organization_UserAccount j ON u.id = j.user_account_id
		WHERE j.customer_organization_id = @customerOrgId
		ORDER BY u.name, u.email`,
		pgx.NamedArgs{"customerOrgId": customerOrganizationID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query users: %w", err)
	} else if result, err := pgx.CollectRows[types.UserAccountWithUserRole](rows, pgx.RowToStructByPos); err != nil {
		return nil, fmt.Errorf("could not map users: %w", err)
	} else {
		return result, nil
	}
}

func GetUserAccountByID(ctx context.Context, id uuid.UUID) (*types.UserAccount, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+userAccountOutputExpr+" FROM UserAccount u WHERE u.id = @id",
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query users: %w", err)
	} else if userAccount, err := pgx.CollectExactlyOneRow[types.UserAccount](rows, pgx.RowToStructByPos); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, fmt.Errorf("could not map user: %w", err)
		}
	} else {
		return &userAccount, nil
	}
}

func GetUserAccountByEmail(ctx context.Context, email string) (*types.UserAccount, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+userAccountOutputExpr+" FROM UserAccount u WHERE u.email = @email",
		pgx.NamedArgs{"email": email},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query users: %w", err)
	} else if userAccount, err := pgx.CollectExactlyOneRow[types.UserAccount](rows, pgx.RowToStructByPos); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, fmt.Errorf("could not map user: %w", err)
		}
	} else {
		return &userAccount, nil
	}
}

func GetUserAccountWithRole(
	ctx context.Context,
	userID, orgID uuid.UUID,
	customerOrgID *uuid.UUID,
) (*types.UserAccountWithUserRole, error) {
	db := internalctx.GetDb(ctx)
	checkCustomerOrgID := customerOrgID != nil
	rows, err := db.Query(ctx,
		"SELECT "+userAccountWithRoleOutputExprWithAlias+`
		FROM UserAccount u
		INNER JOIN Organization_UserAccount j
			ON u.id = j.user_account_id
		WHERE u.id = @id
			AND j.organization_id = @orgId
			AND (NOT @checkCustomerOrgId OR j.customer_organization_id = @customerOrganizationId)`,
		pgx.NamedArgs{
			"id":                     userID,
			"orgId":                  orgID,
			"customerOrganizationId": customerOrgID,
			"checkCustomerOrgId":     checkCustomerOrgID,
		},
	)
	if err != nil {
		return nil, err
	}
	userAccount, err := pgx.CollectExactlyOneRow[types.UserAccountWithUserRole](rows, pgx.RowToStructByPos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		} else {
			return nil, fmt.Errorf("could not map user: %w", err)
		}
	} else {
		return &userAccount, nil
	}
}

func GetUserAccountAndOrg(ctx context.Context, userID, orgID uuid.UUID) (
	*types.UserAccountWithUserRole,
	*types.Organization,
	error,
) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT ("+userAccountWithRoleOutputExpr+`),
					(`+organizationOutputExpr+`)
			FROM UserAccount u
			INNER JOIN Organization_UserAccount j ON u.id = j.user_account_id
			INNER JOIN Organization o ON o.id = j.organization_id
			WHERE u.id = @id
				AND j.organization_id = @orgId
				AND o.deleted_at IS NULL`,
		pgx.NamedArgs{
			"id":    userID,
			"orgId": orgID,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	res, err := pgx.CollectExactlyOneRow[struct {
		User types.UserAccountWithUserRole
		Org  types.Organization
	}](rows, pgx.RowToStructByPos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, apierrors.ErrNotFound
		} else {
			return nil, nil, fmt.Errorf("could not map user or org: %w", err)
		}
	} else {
		return &res.User, &res.Org, nil
	}
}

func GetCustomerAndOrgForDeploymentTarget(
	ctx context.Context,
	id uuid.UUID,
) (*types.CustomerOrganization, *types.Organization, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT
			CASE WHEN co.id IS NOT NULL THEN (`+customerOrganizationOutputExpr+`) END AS customer_organization,
				(`+organizationOutputExpr+`) AS organization
			FROM DeploymentTarget dt
			JOIN Organization o ON o.id = dt.organization_id
			LEFT JOIN CustomerOrganization co ON co.id = dt.customer_organization_id
			WHERE dt.id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, nil, err
	}
	res, err := pgx.CollectExactlyOneRow[struct {
		CustomerOrganization *types.CustomerOrganization
		Org                  types.Organization
	}](rows, pgx.RowToStructByPos)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, apierrors.ErrNotFound
		} else {
			return nil, nil, fmt.Errorf("could not map customer or org: %w", err)
		}
	} else {
		return res.CustomerOrganization, &res.Org, nil
	}
}

func UpdateUserAccountLastLoggedIn(ctx context.Context, userID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`UPDATE UserAccount SET last_logged_in_at = now() WHERE id = @id`,
		pgx.NamedArgs{"id": userID},
	)
	if err == nil && cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("could not update last_logged_in_at on UserAccount: %w", err)
	}
	return err
}

func UpdateUserAccountImage(ctx context.Context, userAccount *types.UserAccountWithUserRole, imageID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	row := db.QueryRow(ctx,
		`UPDATE UserAccount SET image_id = @imageId WHERE id = @id RETURNING image_id`,
		pgx.NamedArgs{"imageId": imageID, "id": userAccount.ID},
	)
	if err := row.Scan(&userAccount.ImageID); err != nil {
		return fmt.Errorf("could not save image id to user account: %w", err)
	}
	return nil
}

func ExistsUserAccountWithEmail(ctx context.Context, email string) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT EXISTS(SELECT 1 FROM UserAccount WHERE email = @email)`,
		pgx.NamedArgs{"email": email},
	)
	if err != nil {
		return false, err
	}
	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, err
	}
	return exists, nil
}

func UpdateUserAccountMFASecret(ctx context.Context, userID uuid.UUID, secret string) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`UPDATE UserAccount SET mfa_secret = @secret WHERE id = @id`,
		pgx.NamedArgs{"secret": secret, "id": userID},
	)
	if err != nil {
		return fmt.Errorf("could not update MFA secret: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}

func EnableUserAccountMFA(ctx context.Context, userID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`UPDATE UserAccount SET mfa_enabled = true, mfa_enabled_at = now() WHERE id = @id`,
		pgx.NamedArgs{"id": userID},
	)
	if err != nil {
		return fmt.Errorf("could not enable MFA: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}

func DisableUserAccountMFA(ctx context.Context, userID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`UPDATE UserAccount SET mfa_enabled = false, mfa_secret = NULL, mfa_enabled_at = NULL WHERE id = @id`,
		pgx.NamedArgs{"id": userID},
	)
	if err != nil {
		return fmt.Errorf("could not disable MFA: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}
