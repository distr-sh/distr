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

const (
	organizationBrandingOutputExpr = `
		b.id, b.created_at, b.organization_id, b.updated_at, b.updated_by_user_account_id, b.title, b.description,
		b.logo_image_id, b.app_domain, b.registry_domain, b.email_from_address
	`
)

func GetOrganizationBranding(ctx context.Context, organizationID uuid.UUID) (*types.OrganizationBranding, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+organizationBrandingOutputExpr+
			"FROM OrganizationBranding b "+
			"WHERE b.organization_id = @organizationId",
		pgx.NamedArgs{"organizationId": organizationID})
	if err != nil {
		return nil, fmt.Errorf("failed to query OrganizationBranding: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.OrganizationBranding])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get OrganizationBranding: %w", err)
	} else {
		return &result, nil
	}
}

func CreateOrganizationBranding(ctx context.Context, b *types.OrganizationBranding) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO OrganizationBranding AS b
			(organization_id, updated_at, updated_by_user_account_id, title, description, logo_image_id)
			VALUES (@organization_id, @updated_at, @updated_by_user_account_id, @title, @description, @logo_image_id)
			RETURNING `+organizationBrandingOutputExpr,
		pgx.NamedArgs{
			"organization_id":            b.OrganizationID,
			"updated_at":                 b.UpdatedAt,
			"updated_by_user_account_id": b.UpdatedByUserAccountID,
			"title":                      b.Title,
			"description":                b.Description,
			"logo_image_id":              b.LogoImageID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create OrganizationBranding: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.OrganizationBranding])
	if err != nil {
		return fmt.Errorf("could not save OrganizationBranding: %w", err)
	} else {
		*b = result
		return nil
	}
}

func UpdateOrganizationBranding(ctx context.Context, b *types.OrganizationBranding) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`UPDATE OrganizationBranding AS b
		SET updated_at = @updated_at,
			updated_by_user_account_id = @updated_by_user_account_id,
			title = @title,
			description = @description,
			logo_image_id = @logo_image_id
		WHERE organization_id = @organization_id
		RETURNING `+organizationBrandingOutputExpr,
		pgx.NamedArgs{
			"organization_id":            b.OrganizationID,
			"updated_at":                 b.UpdatedAt,
			"updated_by_user_account_id": b.UpdatedByUserAccountID,
			"title":                      b.Title,
			"description":                b.Description,
			"logo_image_id":              b.LogoImageID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update OrganizationBranding: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.OrganizationBranding])
	if err != nil {
		return fmt.Errorf("could not save OrganizationBranding: %w", err)
	} else {
		*b = result
		return nil
	}
}
