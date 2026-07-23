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

const customDomainOutputExpr = `
	d.id, d.created_at, d.domain, d.domain_type, d.organization_id,
	d.customer_organization_id, d.partner_organization_id
`

func CreateCustomDomain(ctx context.Context, customDomain *types.CustomDomain) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO CustomDomain AS d (domain, domain_type, organization_id)
		VALUES (@domain, @domainType, @organizationId)
		RETURNING`+customDomainOutputExpr,
		pgx.NamedArgs{
			"domain":         customDomain.Domain,
			"domainType":     customDomain.Type,
			"organizationId": customDomain.OrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert CustomDomain: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomDomain])
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.UniqueViolation {
		return fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
	} else if err != nil {
		return fmt.Errorf("could not collect CustomDomain: %w", err)
	}
	*customDomain = result
	return nil
}

func GetCustomDomains(ctx context.Context, organizationID uuid.UUID) ([]types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customDomainOutputExpr+
			"FROM CustomDomain d WHERE d.organization_id = @organizationId ORDER BY d.created_at, d.domain",
		pgx.NamedArgs{"organizationId": organizationID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomDomains: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.CustomDomain])
	if err != nil {
		return nil, fmt.Errorf("could not collect CustomDomains: %w", err)
	}
	return result, nil
}

// GetOrgWideCustomDomains returns the organization's unscoped (org-wide) custom domains,
// i.e. those not dedicated to a customer or partner organization. At most one row per
// domain type exists (enforced by a partial unique index).
func GetOrgWideCustomDomains(ctx context.Context, organizationID uuid.UUID) ([]types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customDomainOutputExpr+
			`FROM CustomDomain d
			WHERE d.organization_id = @organizationId
				AND d.customer_organization_id IS NULL
				AND d.partner_organization_id IS NULL`,
		pgx.NamedArgs{"organizationId": organizationID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomDomains: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.CustomDomain])
	if err != nil {
		return nil, fmt.Errorf("could not collect CustomDomains: %w", err)
	}
	return result, nil
}

func DeleteCustomDomain(ctx context.Context, id, organizationID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		"DELETE FROM CustomDomain WHERE id = @id AND organization_id = @organizationId",
		pgx.NamedArgs{"id": id, "organizationId": organizationID},
	)
	if err != nil {
		return fmt.Errorf("could not delete CustomDomain: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}

// ExistsCustomDomain reports whether the given (normalized) domain is registered. It backs the
// Caddy on-demand TLS "ask" endpoint and runs during TLS handshakes, so it must stay a single
// indexed lookup (the unique constraint on CustomDomain.domain provides the index).
func ExistsCustomDomain(ctx context.Context, domain string) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT true FROM CustomDomain WHERE domain = @domain",
		pgx.NamedArgs{"domain": domain},
	)
	if err != nil {
		return false, fmt.Errorf("could not query CustomDomain: %w", err)
	}
	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("could not query CustomDomain: %w", err)
	}
	return exists, nil
}

// GetOrganizationBrandingByCustomDomain resolves the branding of the organization owning the
// given (normalized) custom domain. It returns nil when the domain is not registered, and also
// when the owning organization has no branding row yet.
func GetOrganizationBrandingByCustomDomain(ctx context.Context, host string) (*types.OrganizationBranding, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+organizationBrandingOutputExpr+
			` FROM CustomDomain d
			JOIN OrganizationBranding b ON b.organization_id = d.organization_id
			WHERE d.domain = @host`,
		pgx.NamedArgs{"host": host},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query organization branding by custom domain: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.OrganizationBranding])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("could not get organization branding by custom domain: %w", err)
	}
	return &result, nil
}
