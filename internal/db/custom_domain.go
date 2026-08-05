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
	d.id, d.created_at, d.domain, d.domain_type, d.organization_id
`

// CreateCustomDomains inserts all given custom domains with a single statement, so either all
// of them are created or none (e.g. on a unique constraint violation, mapped to ErrConflict).
func CreateCustomDomains(ctx context.Context, customDomains []types.CustomDomain) ([]types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	domains := make([]string, len(customDomains))
	domainTypes := make([]types.DomainType, len(customDomains))
	organizationIDs := make([]uuid.UUID, len(customDomains))
	for i, customDomain := range customDomains {
		domains[i] = customDomain.Domain
		domainTypes[i] = customDomain.Type
		organizationIDs[i] = customDomain.OrganizationID
	}
	rows, err := db.Query(ctx,
		`INSERT INTO CustomDomain AS d (domain, domain_type, organization_id)
		SELECT * FROM unnest(@domains::TEXT[], @domainTypes::CUSTOM_DOMAIN_TYPE[], @organizationIds::UUID[])
		RETURNING`+customDomainOutputExpr,
		pgx.NamedArgs{
			"domains":         domains,
			"domainTypes":     domainTypes,
			"organizationIds": organizationIDs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not insert CustomDomains: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.CustomDomain])
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.UniqueViolation {
		return nil, fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
	} else if err != nil {
		return nil, fmt.Errorf("could not collect CustomDomains: %w", err)
	}
	return result, nil
}

// GetCustomDomains returns the organization's custom domains. There is at most one per domain
// type, enforced by a unique constraint.
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

func DeleteCustomDomain(ctx context.Context, id, organizationID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		"DELETE FROM CustomDomain WHERE id = @id AND organization_id = @organizationId",
		pgx.NamedArgs{"id": id, "organizationId": organizationID},
	)
	if err != nil {
		// A domain an identity provider hangs off is deleted with ON DELETE RESTRICT, so that removing a
		// domain cannot silently take everyone's sign-in with it.
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.ForeignKeyViolation {
			return fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
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

func GetCustomDomainByDomain(ctx context.Context, domain string) (*types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customDomainOutputExpr+"FROM CustomDomain d WHERE d.domain = @domain",
		pgx.NamedArgs{"domain": domain},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomDomain: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomDomain])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("could not get CustomDomain by domain: %w", err)
	}
	return &result, nil
}
