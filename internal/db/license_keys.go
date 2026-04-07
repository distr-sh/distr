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

const licenseKeyOutExpr = `lk.id, lk.created_at, lk.name, lk.description, lr.payload, ` +
	`lr.not_before, lr.expires_at, lr.created_at AS last_revised_at, lk.organization_id, lk.customer_organization_id`

const licenseKeyLatestRevisionJoin = `
	JOIN LATERAL (
		SELECT payload, not_before, expires_at, created_at
		FROM LicenseKeyRevision
		WHERE license_key_id = lk.id
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	) lr ON true`

func GetLicenseKeys(ctx context.Context, orgID uuid.UUID) ([]types.LicenseKey, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+licenseKeyOutExpr+`
		FROM LicenseKey lk`+licenseKeyLatestRevisionJoin+`
		WHERE lk.organization_id = @orgId
		ORDER BY lk.name`,
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
		FROM LicenseKey lk`+licenseKeyLatestRevisionJoin+`
		WHERE lk.organization_id = @orgId AND lk.customer_organization_id = @customerOrgId
		ORDER BY lk.name`,
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
		FROM LicenseKey lk`+licenseKeyLatestRevisionJoin+`
		WHERE lk.id = @id`,
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
	return RunTx(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)
		rows, err := db.Query(ctx, `
			INSERT INTO LicenseKey (name, description, organization_id, customer_organization_id)
			VALUES (@name, @description, @organizationId, @customerOrganizationId)
			RETURNING id`,
			pgx.NamedArgs{
				"name":                   licenseKey.Name,
				"description":            licenseKey.Description,
				"organizationId":         licenseKey.OrganizationID,
				"customerOrganizationId": licenseKey.CustomerOrganizationID,
			},
		)
		if err != nil {
			return fmt.Errorf("could not insert LicenseKey: %w", err)
		}
		var insertedID uuid.UUID
		if id, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[uuid.UUID]); err != nil {
			var pgError *pgconn.PgError
			if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
				return fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
			}
			return err
		} else {
			insertedID = id
		}

		revision := types.LicenseKeyRevision{
			LicenseKeyID: insertedID,
			NotBefore:    licenseKey.NotBefore,
			ExpiresAt:    licenseKey.ExpiresAt,
			Payload:      licenseKey.Payload,
		}
		if err := createLicenseKeyRevision(ctx, &revision); err != nil {
			return err
		}

		result, err := GetLicenseKeyByID(ctx, insertedID)
		if err != nil {
			return err
		}
		*licenseKey = *result
		return nil
	})
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
		FROM updated lk`+licenseKeyLatestRevisionJoin,
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

func GetLicenseKeyRevisions(ctx context.Context, licenseKeyID uuid.UUID) ([]types.LicenseKeyRevision, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT id, created_at, license_key_id, not_before, expires_at, payload
		FROM LicenseKeyRevision
		WHERE license_key_id = @licenseKeyId
		ORDER BY created_at DESC, id DESC`,
		pgx.NamedArgs{"licenseKeyId": licenseKeyID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query LicenseKeyRevision: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.LicenseKeyRevision])
	if err != nil {
		return nil, fmt.Errorf("could not collect LicenseKeyRevision: %w", err)
	}
	return result, nil
}

func CreateLicenseKeyRevision(ctx context.Context, revision *types.LicenseKeyRevision) error {
	return createLicenseKeyRevision(ctx, revision)
}

func createLicenseKeyRevision(ctx context.Context, revision *types.LicenseKeyRevision) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		INSERT INTO LicenseKeyRevision (license_key_id, not_before, expires_at, payload)
		VALUES (@licenseKeyId, @notBefore, @expiresAt, @payload)
		RETURNING id, created_at, license_key_id, not_before, expires_at, payload`,
		pgx.NamedArgs{
			"licenseKeyId": revision.LicenseKeyID,
			"notBefore":    revision.NotBefore,
			"expiresAt":    revision.ExpiresAt,
			"payload":      revision.Payload,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert LicenseKeyRevision: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.LicenseKeyRevision])
	if err != nil {
		return fmt.Errorf("could not collect LicenseKeyRevision: %w", err)
	}
	*revision = result
	return nil
}
