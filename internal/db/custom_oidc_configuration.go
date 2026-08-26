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

const customOIDCConfigurationOutputExpr = `
	c.id, c.created_at, c.updated_at, c.updated_by_user_account_id, c.organization_id, c.custom_domain_id,
	c.name, c.slug, c.enabled, c.issuer, c.client_id, c.client_secret, c.scopes, c.pkce_enabled, c.sp_initiated,
	c.create_unknown_users, c.default_user_role, c.allowed_email_domains
`

func customOIDCConfigurationArgs(c types.CustomOIDCConfiguration) pgx.NamedArgs {
	return pgx.NamedArgs{
		"organizationId":         c.OrganizationID,
		"customDomainId":         c.CustomDomainID,
		"updatedByUserAccountId": c.UpdatedByUserAccountID,
		"name":                   c.Name,
		"slug":                   c.Slug,
		"enabled":                c.Enabled,
		"issuer":                 c.Issuer,
		"clientId":               c.ClientID,
		"clientSecret":           c.ClientSecret,
		"scopes":                 c.Scopes,
		"pkceEnabled":            c.PKCEEnabled,
		"spInitiated":            c.SPInitiated,
		"createUnknownUsers":     c.CreateUnknownUsers,
		"defaultUserRole":        c.DefaultUserRole,
		"allowedEmailDomains":    c.AllowedEmailDomains,
	}
}

func CreateCustomOIDCConfiguration(ctx context.Context, c *types.CustomOIDCConfiguration) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO CustomOIDCConfiguration AS c (
			organization_id, custom_domain_id, updated_by_user_account_id, name, slug, enabled, issuer, client_id,
			client_secret, scopes, pkce_enabled, sp_initiated, create_unknown_users, default_user_role,
			allowed_email_domains
		) VALUES (
			@organizationId, @customDomainId, @updatedByUserAccountId, @name, @slug, @enabled, @issuer, @clientId,
			@clientSecret, @scopes, @pkceEnabled, @spInitiated, @createUnknownUsers, @defaultUserRole,
			@allowedEmailDomains
		) RETURNING`+customOIDCConfigurationOutputExpr,
		customOIDCConfigurationArgs(*c),
	)
	if err != nil {
		return fmt.Errorf("could not insert CustomOIDCConfiguration: %w", err)
	}
	created, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomOIDCConfiguration])
	if err != nil {
		return mapCustomOIDCConfigurationError(err)
	}
	*c = created
	return nil
}

func UpdateCustomOIDCConfiguration(ctx context.Context, c *types.CustomOIDCConfiguration) error {
	db := internalctx.GetDb(ctx)
	args := customOIDCConfigurationArgs(*c)
	args["id"] = c.ID
	rows, err := db.Query(ctx,
		`UPDATE CustomOIDCConfiguration AS c SET
			updated_at = now(),
			updated_by_user_account_id = @updatedByUserAccountId,
			custom_domain_id = @customDomainId,
			name = @name,
			slug = @slug,
			enabled = @enabled,
			issuer = @issuer,
			client_id = @clientId,
			client_secret = @clientSecret,
			scopes = @scopes,
			pkce_enabled = @pkceEnabled,
			sp_initiated = @spInitiated,
			create_unknown_users = @createUnknownUsers,
			default_user_role = @defaultUserRole,
			allowed_email_domains = @allowedEmailDomains
		WHERE c.id = @id AND c.organization_id = @organizationId
		RETURNING`+customOIDCConfigurationOutputExpr,
		args,
	)
	if err != nil {
		return fmt.Errorf("could not update CustomOIDCConfiguration: %w", err)
	}
	updated, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomOIDCConfiguration])
	if errors.Is(err, pgx.ErrNoRows) {
		return apierrors.ErrNotFound
	} else if err != nil {
		return mapCustomOIDCConfigurationError(err)
	}
	*c = updated
	return nil
}

func mapCustomOIDCConfigurationError(err error) error {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		case pgerrcode.ForeignKeyViolation:
			return fmt.Errorf("%w: %w", apierrors.ErrBadRequest, err)
		}
	}
	return fmt.Errorf("could not save CustomOIDCConfiguration: %w", err)
}

// customOIDCConfigurationScopeExpr restricts configurations to the domains of one scope: everything for
// a vendor, one customer's own for a customer, or the domains of the customers assigned to a partner.
const customOIDCConfigurationScopeExpr = `
	JOIN CustomDomain d ON d.id = c.custom_domain_id
		AND (@isVendor
			OR d.customer_organization_id = @customerOrganizationId
			OR EXISTS (
				SELECT 1 FROM CustomerOrganization co
				WHERE co.id = d.customer_organization_id AND co.partner_organization_id = @partnerOrganizationId
			))
`

func customOIDCConfigurationScopeArgs(args pgx.NamedArgs, customerOrgID, partnerOrgID *uuid.UUID) pgx.NamedArgs {
	args["customerOrganizationId"] = customerOrgID
	args["partnerOrganizationId"] = partnerOrgID
	args["isVendor"] = customerOrgID == nil && partnerOrgID == nil
	return args
}

func GetCustomOIDCConfigurations(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrgID, partnerOrgID *uuid.UUID,
) ([]types.CustomOIDCConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customOIDCConfigurationOutputExpr+
			`FROM CustomOIDCConfiguration c`+customOIDCConfigurationScopeExpr+
			`WHERE c.organization_id = @organizationId
			ORDER BY c.name`,
		customOIDCConfigurationScopeArgs(pgx.NamedArgs{"organizationId": organizationID}, customerOrgID, partnerOrgID),
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomOIDCConfiguration: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.CustomOIDCConfiguration])
	if err != nil {
		return nil, fmt.Errorf("could not collect CustomOIDCConfiguration: %w", err)
	}
	return result, nil
}

func GetCustomOIDCConfigurationsForDomain(
	ctx context.Context,
	customDomainID uuid.UUID,
) ([]types.CustomOIDCConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customOIDCConfigurationOutputExpr+
			`FROM CustomOIDCConfiguration c
			WHERE c.custom_domain_id = @customDomainId AND c.enabled
			ORDER BY c.name`,
		pgx.NamedArgs{"customDomainId": customDomainID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomOIDCConfiguration: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.CustomOIDCConfiguration])
	if err != nil {
		return nil, fmt.Errorf("could not collect CustomOIDCConfiguration: %w", err)
	}
	return result, nil
}

func GetCustomOIDCConfigurationOfOrganization(
	ctx context.Context,
	id, organizationID uuid.UUID,
	customerOrgID, partnerOrgID *uuid.UUID,
) (*types.CustomOIDCConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customOIDCConfigurationOutputExpr+
			`FROM CustomOIDCConfiguration c`+customOIDCConfigurationScopeExpr+
			`WHERE c.id = @id AND c.organization_id = @organizationId`,
		customOIDCConfigurationScopeArgs(
			pgx.NamedArgs{"id": id, "organizationId": organizationID}, customerOrgID, partnerOrgID),
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomOIDCConfiguration: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomOIDCConfiguration])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not get CustomOIDCConfiguration: %w", err)
	}
	return &result, nil
}

// GetCustomOIDCConfigurationForHost looks a provider up by the domain of the host the login arrived on,
// because the slug is only unique per domain. The organization slug is matched as well, even though the
// domain already determines the organization, so that the slug in the URL cannot be an arbitrary one.
func GetCustomOIDCConfigurationForHost(
	ctx context.Context,
	customDomainID uuid.UUID,
	organizationSlug, slug string,
) (*types.CustomOIDCConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT"+customOIDCConfigurationOutputExpr+
			`FROM CustomOIDCConfiguration c
			JOIN Organization o ON o.id = c.organization_id
			WHERE c.custom_domain_id = @customDomainId AND c.slug = @slug
				AND o.slug = @organizationSlug AND o.deleted_at IS NULL`,
		pgx.NamedArgs{"customDomainId": customDomainID, "organizationSlug": organizationSlug, "slug": slug},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query CustomOIDCConfiguration: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.CustomOIDCConfiguration])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not get CustomOIDCConfiguration: %w", err)
	}
	return &result, nil
}

func DeleteCustomOIDCConfiguration(
	ctx context.Context,
	id, organizationID uuid.UUID,
	customerOrgID, partnerOrgID *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`DELETE FROM CustomOIDCConfiguration c
		USING CustomDomain d
		WHERE c.id = @id AND c.organization_id = @organizationId AND d.id = c.custom_domain_id
			AND (@isVendor
				OR d.customer_organization_id = @customerOrganizationId
				OR EXISTS (
					SELECT 1 FROM CustomerOrganization co
					WHERE co.id = d.customer_organization_id AND co.partner_organization_id = @partnerOrganizationId
				))`,
		customOIDCConfigurationScopeArgs(
			pgx.NamedArgs{"id": id, "organizationId": organizationID}, customerOrgID, partnerOrgID),
	)
	if err != nil {
		return fmt.Errorf("could not delete CustomOIDCConfiguration: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}
