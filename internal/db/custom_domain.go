package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const customDomainOutputExpr = `
	d.id, d.created_at, d.domain, d.domain_type, d.organization_id, d.customer_organization_id,
	d.verified_at, d.verification_checked_at, d.verification_error
`

// CreateCustomDomains inserts with a single statement so that a conflict on any one domain leaves
// none of them created.
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

// GetCustomDomains backs the internal customdomains resolvers. A nil customerOrgID must not return
// customer rows, or a customer hostname would end up in the vendor's app and registry URLs, so
// GetCustomDomainsForScope — which answers what a caller may see — must never be substituted here.
//
// It returns unverified domains as well. Build outbound URLs through the internal/customdomains
// resolvers, which drop those, and never from this function directly.
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

// GetCustomDomainsForScope lists the custom domains a caller may see. It is the caller-facing
// counterpart to GetCustomDomains, which resolves vendor-only hostnames for links and manifests
// and must never be widened the same way.
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

func GetCustomDomainOfOrganization(
	ctx context.Context,
	id, organizationID uuid.UUID,
	customerOrgID, partnerOrgID *uuid.UUID,
) (*types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customDomainOutputExpr+
			`FROM CustomDomain d
			WHERE d.id = @id AND d.organization_id = @organizationId
				AND (@isVendor
					OR d.customer_organization_id = @customerOrganizationId
					OR EXISTS (
						SELECT 1 FROM CustomerOrganization c
						WHERE c.id = d.customer_organization_id AND c.partner_organization_id = @partnerOrganizationId
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
		return nil, fmt.Errorf("could not query CustomDomain: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomDomain])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not get CustomDomain: %w", err)
	}
	return &result, nil
}

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

// ExistsCustomDomain expects an already normalized domain. It runs during TLS handshakes (see
// TLSAskHandler), so it must stay an index lookup, and it excludes soft-deleted organizations,
// whose domains would otherwise keep getting certificates issued.
func ExistsCustomDomain(ctx context.Context, domain string) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT true FROM CustomDomain d
		JOIN Organization o ON o.id = d.organization_id
		WHERE d.domain = @domain AND o.deleted_at IS NULL`,
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

// GetCustomDomainsDueForVerification lists the domains the verification job has to check. A domain
// that is currently usable is only rechecked once its last check has aged past refreshAfter, whereas
// one that is failing or has never been verified is due on every run: an organization that has just
// fixed its record would otherwise stay on the fallback host for a whole refresh interval.
func GetCustomDomainsDueForVerification(
	ctx context.Context,
	refreshAfter time.Duration,
) ([]types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customDomainOutputExpr+
			`FROM CustomDomain d
			JOIN Organization o ON o.id = d.organization_id
			WHERE o.deleted_at IS NULL
				AND (d.verified_at IS NULL
					OR d.verification_error IS NOT NULL
					OR d.verification_checked_at IS NULL
					OR now() - d.verification_checked_at > @refreshAfter)
			ORDER BY d.verification_checked_at NULLS FIRST`,
		pgx.NamedArgs{"refreshAfter": refreshAfter},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomDomains due for verification: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.CustomDomain])
	if err != nil {
		return nil, fmt.Errorf("could not collect CustomDomains: %w", err)
	}
	return result, nil
}

// SetCustomDomainVerificationResult records a completed check. A nil verificationError marks the
// domain verified; a non-nil one is the reason its record is wrong. A lookup that did not complete
// is not a result at all and must not be passed here, see SetCustomDomainVerificationAttempted.
func SetCustomDomainVerificationResult(
	ctx context.Context,
	id uuid.UUID,
	verificationError *string,
) (*types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE CustomDomain AS d SET
			verification_checked_at = now(),
			verification_error = @verificationError,
			verified_at = CASE WHEN @verificationError::TEXT IS NULL THEN now() ELSE verified_at END
		WHERE d.id = @id
		RETURNING`+customDomainOutputExpr,
		pgx.NamedArgs{"id": id, "verificationError": verificationError},
	)
	if err != nil {
		return nil, fmt.Errorf("could not update CustomDomain verification result: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomDomain])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not update CustomDomain verification result: %w", err)
	}
	return &result, nil
}

// SetCustomDomainVerificationAttempted records that a check ran without reaching a conclusion, so
// that the job's schedule moves on while the domain keeps whatever state it was in.
func SetCustomDomainVerificationAttempted(ctx context.Context, id uuid.UUID) (*types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE CustomDomain AS d SET verification_checked_at = now()
		WHERE d.id = @id
		RETURNING`+customDomainOutputExpr,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not update CustomDomain verification attempt: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomDomain])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not update CustomDomain verification attempt: %w", err)
	}
	return &result, nil
}

func GetCustomDomainByDomain(ctx context.Context, domain string) (*types.CustomDomain, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customDomainOutputExpr+
			`FROM CustomDomain d
			JOIN Organization o ON o.id = d.organization_id
			WHERE d.domain = @domain AND o.deleted_at IS NULL`,
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
