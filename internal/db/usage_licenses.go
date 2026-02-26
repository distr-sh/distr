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

const usageLicenseOutExpr = `ul.id, ul.created_at, ul.name, ul.description, ul.payload, ul.token, ` +
	`ul.not_before, ul.expires_at, ul.organization_id, ul.customer_organization_id `

func GetUsageLicenses(ctx context.Context, orgID uuid.UUID) ([]types.UsageLicense, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+usageLicenseOutExpr+`
		FROM UsageLicense ul
		WHERE ul.organization_id = @orgId
		ORDER BY ul.name`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query UsageLicense: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.UsageLicense])
	if err != nil {
		return nil, fmt.Errorf("could not query UsageLicense: %w", err)
	}
	return result, nil
}

func GetUsageLicensesByCustomerOrgID(
	ctx context.Context, customerOrgID, orgID uuid.UUID,
) ([]types.UsageLicense, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+usageLicenseOutExpr+`
		FROM UsageLicense ul
		WHERE ul.organization_id = @orgId AND ul.customer_organization_id = @customerOrgId
		ORDER BY ul.name`,
		pgx.NamedArgs{"orgId": orgID, "customerOrgId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query UsageLicense: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.UsageLicense])
	if err != nil {
		return nil, fmt.Errorf("could not query UsageLicense: %w", err)
	}
	return result, nil
}

func GetUsageLicenseByID(ctx context.Context, id uuid.UUID) (*types.UsageLicense, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+usageLicenseOutExpr+`
		FROM UsageLicense ul
		WHERE ul.id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query UsageLicense: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.UsageLicense]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not collect UsageLicense: %w", err)
	} else {
		return &result, nil
	}
}

func CreateUsageLicense(ctx context.Context, license *types.UsageLicense) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		WITH inserted AS (
			INSERT INTO UsageLicense (
				name, description, payload, token, not_before, expires_at,
				organization_id, customer_organization_id
			) VALUES (
				@name, @description, @payload, @token, @notBefore, @expiresAt,
				@organizationId, @customerOrganizationId
			) RETURNING *
		)
		SELECT `+usageLicenseOutExpr+`
		FROM inserted ul`,
		pgx.NamedArgs{
			"name":                   license.Name,
			"description":            license.Description,
			"payload":                license.Payload,
			"token":                  license.Token,
			"notBefore":              license.NotBefore,
			"expiresAt":              license.ExpiresAt,
			"organizationId":         license.OrganizationID,
			"customerOrganizationId": license.CustomerOrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert UsageLicense: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.UsageLicense]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*license = result
		return nil
	}
}

func UpdateUsageLicenseMetadata(
	ctx context.Context, id uuid.UUID, name string, description *string,
) (*types.UsageLicense, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		WITH updated AS (
			UPDATE UsageLicense SET
				name = @name,
				description = @description
			WHERE id = @id RETURNING *
		)
		SELECT `+usageLicenseOutExpr+`
		FROM updated ul`,
		pgx.NamedArgs{
			"id":          id,
			"name":        name,
			"description": description,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not update UsageLicense: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.UsageLicense]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return nil, err
	} else {
		return &result, nil
	}
}

func DeleteUsageLicenseWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `DELETE FROM UsageLicense WHERE id = @id`, pgx.NamedArgs{"id": id})
	if err != nil {
		return fmt.Errorf("could not delete UsageLicense: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("could not delete UsageLicense: %w", apierrors.ErrNotFound)
	}
	return nil
}
