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
		b.logo_image_id, b.app_domain, b.registry_domain, b.email_from_address, b.page_title, b.favicon_image_id
	`
)

func GetOrganizationBranding(ctx context.Context, organizationID uuid.UUID) (*types.OrganizationBranding, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+organizationBrandingOutputExpr+
			" FROM OrganizationBranding b "+
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

func UpsertOrganizationBranding(ctx context.Context, b *types.OrganizationBranding) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO OrganizationBranding AS b
			(organization_id, updated_at, updated_by_user_account_id, title, description, logo_image_id,
				page_title, favicon_image_id)
			VALUES (@organization_id, now(), @updated_by_user_account_id, @title, @description, @logo_image_id,
				@page_title, @favicon_image_id)
			ON CONFLICT (organization_id) DO UPDATE SET
				updated_at = now(),
				updated_by_user_account_id = @updated_by_user_account_id,
				title = @title,
				description = @description,
				logo_image_id = @logo_image_id,
				page_title = @page_title,
				favicon_image_id = @favicon_image_id
			RETURNING `+organizationBrandingOutputExpr,
		pgx.NamedArgs{
			"organization_id":            b.OrganizationID,
			"updated_by_user_account_id": b.UpdatedByUserAccountID,
			"title":                      b.Title,
			"description":                b.Description,
			"logo_image_id":              b.LogoImageID,
			"page_title":                 b.PageTitle,
			"favicon_image_id":           b.FaviconImageID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to upsert OrganizationBranding: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.OrganizationBranding])
	if err != nil {
		return fmt.Errorf("could not save OrganizationBranding: %w", err)
	} else {
		*b = result
		return nil
	}
}

// OrganizationLegacyBrandingDomains holds an organization's legacy branding domain columns. It is used to
// check self-service CustomDomain registrations for conflicts with legacy domains of other organizations.
type OrganizationLegacyBrandingDomains struct {
	OrganizationID uuid.UUID `db:"organization_id"`
	AppDomain      *string   `db:"app_domain"`
	RegistryDomain *string   `db:"registry_domain"`
}

// GetOrganizationLegacyBrandingDomains returns values that are not guaranteed to be normalized
// (legacy app domains may contain a scheme), so callers must normalize them before comparing.
func GetOrganizationLegacyBrandingDomains(ctx context.Context) ([]OrganizationLegacyBrandingDomains, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT b.organization_id, b.app_domain, b.registry_domain FROM OrganizationBranding b
			JOIN Organization o ON o.id = b.organization_id
			WHERE o.deleted_at IS NULL AND (b.app_domain IS NOT NULL OR b.registry_domain IS NOT NULL)`,
	)
	if err != nil {
		return nil, fmt.Errorf("could not query legacy branding domains: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[OrganizationLegacyBrandingDomains])
	if err != nil {
		return nil, fmt.Errorf("could not collect legacy branding domains: %w", err)
	}
	return result, nil
}

// GetOrganizationBrandingByAppDomain requires both the stored app_domain and the given host to be
// normalized (lower-case, without scheme or port).
func GetOrganizationBrandingByAppDomain(ctx context.Context, host string) (*types.OrganizationBranding, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+organizationBrandingOutputExpr+
			` FROM OrganizationBranding b
			JOIN Organization o ON o.id = b.organization_id
			WHERE b.app_domain = @host AND o.deleted_at IS NULL`,
		pgx.NamedArgs{"host": host},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query organization branding by app domain: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.OrganizationBranding])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("could not get organization branding by app domain: %w", err)
	}

	return &result, nil
}
