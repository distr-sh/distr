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
	d.id, d.created_at, d.domain, d.domain_type, d.organization_id, d.customer_organization_id
`

// CreateCustomDomains inserts all given custom domains with a single statement, so either all
// of them are created or none (e.g. on a unique constraint violation, mapped to ErrConflict).
func CreateCustomDomains(ctx context.Context, customDomains []types.CustomDomain) ([]types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	domains := make([]string, len(customDomains))
	domainTypes := make([]types.DomainType, len(customDomains))
	organizationIDs := make([]uuid.UUID, len(customDomains))
	customerOrganizationIDs := make([]*uuid.UUID, len(customDomains))
	for i, customDomain := range customDomains {
		domains[i] = customDomain.Domain
		domainTypes[i] = customDomain.Type
		organizationIDs[i] = customDomain.OrganizationID
		customerOrganizationIDs[i] = customDomain.CustomerOrganizationID
	}
	rows, err := db.Query(ctx,
		`INSERT INTO CustomDomain AS d (domain, domain_type, organization_id, customer_organization_id)
		SELECT * FROM unnest(@domains::TEXT[], @domainTypes::CUSTOM_DOMAIN_TYPE[], @organizationIds::UUID[],
			@customerOrganizationIds::UUID[])
		RETURNING`+customDomainOutputExpr,
		pgx.NamedArgs{
			"domains":                 domains,
			"domainTypes":             domainTypes,
			"organizationIds":         organizationIDs,
			"customerOrganizationIds": customerOrganizationIDs,
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

// GetCustomDomains returns the custom domains of one scope: the vendor's own rows when customerOrgID
// is nil, otherwise that customer's rows. A nil customerOrgID must not return customer rows, or a
// customer hostname would end up in the vendor's app and registry URLs. Used only by the internal
// customdomains resolvers; a caller-facing listing belongs in GetCustomDomainsForScope instead, which
// answers a different question (what may this caller see) and must not be substituted here.
func GetCustomDomains(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrgID *uuid.UUID,
) ([]types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customDomainOutputExpr+
			`FROM CustomDomain d
			WHERE d.organization_id = @organizationId
				AND (CASE WHEN @checkCustomerOrgId
					THEN d.customer_organization_id = @customerOrganizationId
					ELSE d.customer_organization_id IS NULL END)
			ORDER BY d.created_at, d.domain`,
		pgx.NamedArgs{
			"organizationId":         organizationID,
			"customerOrganizationId": customerOrgID,
			"checkCustomerOrgId":     customerOrgID != nil,
		},
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

// GetCustomDomainsForScope lists the custom domains a caller may see: everything in the organization
// for a vendor (own domains and every customer's), one customer's own for a customer, and the domains
// of the customers assigned to a partner for a partner. This is the caller-facing counterpart to
// GetCustomDomains, which is instead used internally to resolve vendor-only hostnames for links and
// manifests and must never be widened the same way.
func GetCustomDomainsForScope(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrgID, partnerOrgID *uuid.UUID,
) ([]types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customDomainOutputExpr+
			`FROM CustomDomain d
			WHERE d.organization_id = @organizationId
				AND (@isVendor
					OR d.customer_organization_id = @customerOrganizationId
					OR EXISTS (
						SELECT 1 FROM CustomerOrganization c
						WHERE c.id = d.customer_organization_id AND c.partner_organization_id = @partnerOrganizationId
					))
			ORDER BY d.created_at, d.domain`,
		pgx.NamedArgs{
			"organizationId":         organizationID,
			"customerOrganizationId": customerOrgID,
			"partnerOrganizationId":  partnerOrgID,
			"isVendor":               customerOrgID == nil && partnerOrgID == nil,
		},
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

// DeleteCustomDomain removes a domain within the caller's scope: any domain for a vendor, own domain
// for a customer, or a domain of an assigned customer for a partner.
func DeleteCustomDomain(
	ctx context.Context,
	id, organizationID uuid.UUID,
	customerOrgID, partnerOrgID *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`DELETE FROM CustomDomain
		WHERE id = @id AND organization_id = @organizationId
			AND (@isVendor
				OR customer_organization_id = @customerOrganizationId
				OR EXISTS (
					SELECT 1 FROM CustomerOrganization c
					WHERE c.id = customer_organization_id AND c.partner_organization_id = @partnerOrganizationId
				))`,
		pgx.NamedArgs{
			"id":                     id,
			"organizationId":         organizationID,
			"customerOrganizationId": customerOrgID,
			"partnerOrganizationId":  partnerOrgID,
			"isVendor":               customerOrgID == nil && partnerOrgID == nil,
		},
	)
	if err != nil {
		// A domain an identity provider hangs off references it with ON DELETE RESTRICT, so that removing
		// a domain cannot silently take everyone's sign-in with it.
		if isStillReferencedError(err) {
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
