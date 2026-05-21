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

const serviceAccountOutputExpr = `
	sa.id, sa.created_at, sa.organization_id, sa.customer_organization_id, sa.name, sa.account_role
`

func CreateServiceAccount(ctx context.Context, sa *types.ServiceAccount) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`INSERT INTO ServiceAccount AS sa (organization_id, customer_organization_id, name, account_role)
			VALUES (@orgId, @customerOrgId, @name, @accountRole)
			RETURNING %s`,
			serviceAccountOutputExpr,
		),
		pgx.NamedArgs{
			"orgId":         sa.OrganizationID,
			"customerOrgId": sa.CustomerOrganizationID,
			"name":          sa.Name,
			"accountRole":   sa.AccountRole,
		},
	)
	if err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("service account %q can not be created: %w", sa.Name, apierrors.ErrAlreadyExists)
		}
		return fmt.Errorf("could not create service account: %w", err)
	}
	if res, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ServiceAccount]); err != nil {
		return fmt.Errorf("could not create service account: %w", err)
	} else {
		*sa = res
		return nil
	}
}

func UpdateServiceAccount(ctx context.Context, sa *types.ServiceAccount) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`UPDATE ServiceAccount AS sa
			SET name = @name, account_role = @accountRole
			WHERE id = @id AND organization_id = @orgId
			RETURNING %s`,
			serviceAccountOutputExpr,
		),
		pgx.NamedArgs{
			"id":          sa.ID,
			"orgId":       sa.OrganizationID,
			"name":        sa.Name,
			"accountRole": sa.AccountRole,
		},
	)
	if err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("service account %q can not be updated: %w", sa.Name, apierrors.ErrAlreadyExists)
		}
		return fmt.Errorf("could not update service account: %w", err)
	}
	if res, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ServiceAccount]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apierrors.ErrNotFound
		}
		return fmt.Errorf("could not update service account: %w", err)
	} else {
		*sa = res
		return nil
	}
}

func DeleteServiceAccount(ctx context.Context, id, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM ServiceAccount WHERE id = @id AND organization_id = @orgId`,
		pgx.NamedArgs{"id": id, "orgId": orgID},
	)
	if err != nil {
		return fmt.Errorf("could not delete service account: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}

func GetServiceAccountsByOrgID(ctx context.Context, orgID uuid.UUID) ([]types.ServiceAccount, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`SELECT %s
			FROM ServiceAccount sa
			WHERE sa.organization_id = @orgId AND sa.customer_organization_id IS NULL
			ORDER BY sa.name`,
			serviceAccountOutputExpr,
		),
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query service accounts: %w", err)
	}
	if result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ServiceAccount]); err != nil {
		return nil, fmt.Errorf("could not map service accounts: %w", err)
	} else {
		return result, nil
	}
}

func GetServiceAccountsByCustomerOrgID(
	ctx context.Context,
	customerOrgID uuid.UUID,
) ([]types.ServiceAccount, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`SELECT %s
			FROM ServiceAccount sa
			WHERE sa.customer_organization_id = @customerOrgId
			ORDER BY sa.name`,
			serviceAccountOutputExpr,
		),
		pgx.NamedArgs{"customerOrgId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query service accounts: %w", err)
	}
	if result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ServiceAccount]); err != nil {
		return nil, fmt.Errorf("could not map service accounts: %w", err)
	} else {
		return result, nil
	}
}

func GetServiceAccountByID(ctx context.Context, id, orgID uuid.UUID) (*types.ServiceAccount, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(
			`SELECT %s
			FROM ServiceAccount sa
			WHERE sa.id = @id AND sa.organization_id = @orgId`,
			serviceAccountOutputExpr,
		),
		pgx.NamedArgs{"id": id, "orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query service account: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ServiceAccount]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not map service account: %w", err)
	} else {
		return &result, nil
	}
}
