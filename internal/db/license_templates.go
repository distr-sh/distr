package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const licenseTemplateOutExpr = `id, created_at, name, organization_id, payload_template, expiration_grace_period_days`

func GetLicenseTemplates(ctx context.Context, orgID uuid.UUID) ([]types.LicenseTemplate, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT `+licenseTemplateOutExpr+` FROM LicenseTemplate WHERE organization_id = @orgId ORDER BY name`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query LicenseTemplate: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.LicenseTemplate])
	if err != nil {
		return nil, fmt.Errorf("could not collect LicenseTemplate: %w", err)
	}
	return result, nil
}

func GetLicenseTemplateByID(ctx context.Context, id, orgID uuid.UUID) (*types.LicenseTemplate, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT `+licenseTemplateOutExpr+` FROM LicenseTemplate WHERE id = @id AND organization_id = @orgId`,
		pgx.NamedArgs{"id": id, "orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query LicenseTemplate: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.LicenseTemplate])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not collect LicenseTemplate: %w", err)
	}
	return &result, nil
}

func CreateLicenseTemplate(ctx context.Context, t *types.LicenseTemplate) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO LicenseTemplate (name, organization_id, payload_template, expiration_grace_period_days)
		VALUES (@name, @orgId, @payloadTemplate, @gracePeriodDays)
		RETURNING `+licenseTemplateOutExpr,
		pgx.NamedArgs{
			"name":            t.Name,
			"orgId":           t.OrganizationID,
			"payloadTemplate": t.PayloadTemplate,
			"gracePeriodDays": t.ExpirationGracePeriodDays,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert LicenseTemplate: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.LicenseTemplate])
	if err != nil {
		return fmt.Errorf("could not collect LicenseTemplate: %w", err)
	}
	*t = result
	return nil
}

func UpdateLicenseTemplate(ctx context.Context, t *types.LicenseTemplate) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE LicenseTemplate SET
			name = @name,
			payload_template = @payloadTemplate,
			expiration_grace_period_days = @gracePeriodDays
		WHERE id = @id AND organization_id = @orgId
		RETURNING `+licenseTemplateOutExpr,
		pgx.NamedArgs{
			"id":              t.ID,
			"orgId":           t.OrganizationID,
			"name":            t.Name,
			"payloadTemplate": t.PayloadTemplate,
			"gracePeriodDays": t.ExpirationGracePeriodDays,
		},
	)
	if err != nil {
		return fmt.Errorf("could not update LicenseTemplate: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.LicenseTemplate])
	if errors.Is(err, pgx.ErrNoRows) {
		return apierrors.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("could not collect LicenseTemplate: %w", err)
	}
	*t = result
	return nil
}

func DeleteLicenseTemplateByID(ctx context.Context, id, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`DELETE FROM LicenseTemplate WHERE id = @id AND organization_id = @orgId`,
		pgx.NamedArgs{"id": id, "orgId": orgID},
	)
	if err != nil {
		return fmt.Errorf("could not delete LicenseTemplate: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}
