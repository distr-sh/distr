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

const licenseKeyOutExpr = `ul.id, ul.created_at, ul.name, ul.description, ul.payload, ul.token, ` +
	`ul.not_before, ul.expires_at, ul.organization_id, ul.customer_organization_id `

func GetLicenseKeys(ctx context.Context, orgID uuid.UUID) ([]types.LicenseKey, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+licenseKeyOutExpr+`
		FROM LicenseKey ul
		WHERE ul.organization_id = @orgId
		ORDER BY ul.name`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query LicenseKey: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.LicenseKey])
	if err != nil {
		return nil, fmt.Errorf("could not query LicenseKey: %w", err)
	}
	return result, nil
}

func GetLicenseKeysByCustomerOrgID(
	ctx context.Context, customerOrgID, orgID uuid.UUID,
) ([]types.LicenseKey, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+licenseKeyOutExpr+`
		FROM LicenseKey ul
		WHERE ul.organization_id = @orgId AND ul.customer_organization_id = @customerOrgId
		ORDER BY ul.name`,
		pgx.NamedArgs{"orgId": orgID, "customerOrgId": customerOrgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query LicenseKey: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.LicenseKey])
	if err != nil {
		return nil, fmt.Errorf("could not query LicenseKey: %w", err)
	}
	return result, nil
}

func GetLicenseKeyByID(ctx context.Context, id uuid.UUID) (*types.LicenseKey, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+licenseKeyOutExpr+`
		FROM LicenseKey ul
		WHERE ul.id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query LicenseKey: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.LicenseKey]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not collect LicenseKey: %w", err)
	} else {
		return &result, nil
	}
}

func CreateLicenseKey(ctx context.Context, licenseKey *types.LicenseKey) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		WITH inserted AS (
			INSERT INTO LicenseKey (
				name, description, payload, token, not_before, expires_at,
				organization_id, customer_organization_id
			) VALUES (
				@name, @description, @payload, @token, @notBefore, @expiresAt,
				@organizationId, @customerOrganizationId
			) RETURNING *
		)
		SELECT `+licenseKeyOutExpr+`
		FROM inserted ul`,
		pgx.NamedArgs{
			"name":                   licenseKey.Name,
			"description":            licenseKey.Description,
			"payload":                licenseKey.Payload,
			"token":                  licenseKey.Token,
			"notBefore":              licenseKey.NotBefore,
			"expiresAt":              licenseKey.ExpiresAt,
			"organizationId":         licenseKey.OrganizationID,
			"customerOrganizationId": licenseKey.CustomerOrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert LicenseKey: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.LicenseKey]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*licenseKey = result
		return nil
	}
}

func UpdateLicenseKeyMetadata(
	ctx context.Context, id uuid.UUID, name string, description *string,
) (*types.LicenseKey, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		WITH updated AS (
			UPDATE LicenseKey SET
				name = @name,
				description = @description
			WHERE id = @id RETURNING *
		)
		SELECT `+licenseKeyOutExpr+`
		FROM updated ul`,
		pgx.NamedArgs{
			"id":          id,
			"name":        name,
			"description": description,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not update LicenseKey: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.LicenseKey]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return nil, err
	} else {
		return &result, nil
	}
}

func UpdateLicenseKeyToken(ctx context.Context, id uuid.UUID, token string) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `UPDATE LicenseKey SET token = @token WHERE id = @id`,
		pgx.NamedArgs{"id": id, "token": token},
	)
	if err != nil {
		return fmt.Errorf("could not update LicenseKey token: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("could not update LicenseKey token: %w", apierrors.ErrNotFound)
	}
	return nil
}

func DeleteLicenseKeyWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `DELETE FROM LicenseKey WHERE id = @id`, pgx.NamedArgs{"id": id})
	if err != nil {
		return fmt.Errorf("could not delete LicenseKey: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("could not delete LicenseKey: %w", apierrors.ErrNotFound)
	}
	return nil
}
