package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	organizationOutputExpr = `
		o.id,
		o.created_at,
		o.name,
		o.slug,
		o.features,
		o.app_domain,
		o.registry_domain,
		o.email_from_address,
		o.subscription_type,
		o.subscription_period,
		o.subscription_ends_at,
		o.stripe_customer_id,
		o.stripe_subscription_id,
		o.subscription_customer_organization_quantity,
		o.subscription_user_account_quantity,
		o.pre_connect_script,
		o.post_connect_script,
		o.connect_script_is_sudo
	`
	organizationWithUserRoleOutputExpr = organizationOutputExpr + `,
		j.user_role,
		cu.id AS customer_organization_id,
		cu.name AS customer_organization_name,
		j.created_at as joined_org_at `
)

func CreateOrganization(ctx context.Context, org *types.Organization) error {
	if buildconfig.IsCommunityEdition() {
		org.SubscriptionType = types.SubscriptionTypeCommunity
		org.Features = []types.Feature{}
	} else {
		org.SubscriptionType = types.SubscriptionTypeTrial
		org.Features = []types.Feature{types.FeatureLicensing}
	}

	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO Organization AS o (name, slug, subscription_type, features)
		VALUES (@name, @slug, @subscription_type, @features)
		RETURNING `+organizationOutputExpr,
		pgx.NamedArgs{
			"name":              org.Name,
			"slug":              org.Slug,
			"subscription_type": org.SubscriptionType,
			"features":          org.Features,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create orgnization: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[types.Organization])
	if err != nil {
		return err
	} else {
		*org = *result
		return nil
	}
}

func UpdateOrganization(ctx context.Context, org *types.Organization) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE Organization AS o
		SET
			name = @name,
			slug = @slug,
			features = @features,
			subscription_type = @subscription_type,
			subscription_period = @subscription_period,
			subscription_ends_at = @subscription_ends_at,
			stripe_customer_id = @stripe_customer_id,
			stripe_subscription_id = @stripe_subscription_id,
			subscription_customer_organization_quantity = @subscription_customer_organization_quantity,
			subscription_user_account_quantity = @subscription_user_account_quantity,
			pre_connect_script = @pre_connect_script,
			post_connect_script = @post_connect_script,
			connect_script_is_sudo = @connect_script_is_sudo
		WHERE id = @id
		RETURNING `+organizationOutputExpr,
		pgx.NamedArgs{
			"id":                     org.ID,
			"name":                   org.Name,
			"features":               org.Features,
			"slug":                   org.Slug,
			"subscription_type":      org.SubscriptionType,
			"subscription_period":    org.SubscriptionPeriod,
			"subscription_ends_at":   org.SubscriptionEndsAt.UTC(),
			"stripe_customer_id":     org.StripeCustomerID,
			"stripe_subscription_id": org.StripeSubscriptionID,
			"subscription_customer_organization_quantity": org.SubscriptionCustomerOrganizationQty,
			"subscription_user_account_quantity":          org.SubscriptionUserAccountQty,
			"pre_connect_script":                          org.PreConnectScript,
			"post_connect_script":                         org.PostConnectScript,
			"connect_script_is_sudo":                      org.ConnectScriptIsSudo,
		},
	)
	if err != nil {
		return err
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[types.Organization]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*org = *result
		return nil
	}
}

func UpdateOrganizationSubscriptionType(ctx context.Context, subscriptionType types.SubscriptionType) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		`UPDATE Organization
		SET subscription_type = @subscription_type`,
		pgx.NamedArgs{"subscription_type": subscriptionType},
	)
	if err != nil {
		return fmt.Errorf("could no update Organization: %w", err)
	}
	return nil
}

func UpdateOrganizationFeaturesWithSubscriptionType(
	ctx context.Context,
	subscriptionType []types.SubscriptionType,
	features []types.Feature,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		`UPDATE Organization
		SET features = @features
		WHERE subscription_type = ANY(@subscription_type)`,
		pgx.NamedArgs{"subscription_type": subscriptionType, "features": features},
	)
	if err != nil {
		return fmt.Errorf("could no update Organization: %w", err)
	}
	return nil
}

func GetOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]types.OrganizationWithUserRole, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT`+organizationWithUserRoleOutputExpr+`
			FROM UserAccount u
			INNER JOIN Organization_UserAccount j ON u.id = j.user_account_id
			INNER JOIN Organization o ON o.id = j.organization_id
			LEFT JOIN CustomerOrganization cu ON cu.id = j.customer_organization_id
			WHERE u.id = @id
				AND o.deleted_at IS NULL
			ORDER BY o.created_at
	`, pgx.NamedArgs{"id": userID})
	if err != nil {
		return nil, err
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.OrganizationWithUserRole])
	if err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

func GetAllOrganizationsForSuperAdmin(ctx context.Context) ([]types.OrganizationWithUserRole, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT`+organizationOutputExpr+`,
			'' as user_role,
			NULL::UUID as customer_organization_id,
			NULL::TEXT as customer_organization_name,
			o.created_at as joined_org_at
			FROM Organization o
			WHERE o.deleted_at IS NULL
			ORDER BY o.created_at
	`)
	if err != nil {
		return nil, err
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.OrganizationWithUserRole])
	if err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

func GetOrganizationByID(ctx context.Context, orgID uuid.UUID) (*types.Organization, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"SELECT "+organizationOutputExpr+" FROM Organization o WHERE id = @id AND o.deleted_at IS NULL",
		pgx.NamedArgs{"id": orgID},
	)
	if err != nil {
		return nil, err
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Organization])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, err
	} else {
		return &result, nil
	}
}

func GetOrganizationWithBranding(ctx context.Context, orgID uuid.UUID) (*types.OrganizationWithBranding, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		fmt.Sprintf(
			`SELECT `+organizationOutputExpr+`,
				CASE WHEN b.id IS NOT NULL THEN (%v) END AS branding
			FROM Organization o
			LEFT JOIN OrganizationBranding b ON b.organization_id = o.id
			WHERE o.id = @id
				AND o.deleted_at IS NULL`,
			organizationBrandingOutputExpr,
		),
		pgx.NamedArgs{"id": orgID},
	)
	if err != nil {
		return nil, err
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[types.OrganizationWithBranding])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not get organization: %w", err)
	} else {
		return result, nil
	}
}

func SetOrganizationDeletedAtNow(ctx context.Context, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		"UPDATE Organization SET deleted_at = now() WHERE id = @id AND deleted_at IS NULL",
		pgx.NamedArgs{"id": orgID},
	)
	if err != nil {
		return fmt.Errorf("could not update Organization: %w", err)
	}
	return nil
}
