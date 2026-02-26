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

// Configuration

func GetSupportBundleConfiguration(ctx context.Context, orgID uuid.UUID) (*types.SupportBundleConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT id, created_at, organization_id
		FROM SupportBundleConfiguration
		WHERE organization_id = @orgId`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query support bundle configuration: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SupportBundleConfiguration])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not get support bundle configuration: %w", err)
	}
	return &result, nil
}

func GetSupportBundleConfigurationEnvVars(
	ctx context.Context, configID uuid.UUID,
) ([]types.SupportBundleConfigurationEnvVar, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT id, support_bundle_configuration_id, name, redacted
		FROM SupportBundleConfigurationEnvVar
		WHERE support_bundle_configuration_id = @configId
		ORDER BY name`,
		pgx.NamedArgs{"configId": configID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query support bundle config env vars: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.SupportBundleConfigurationEnvVar])
	if err != nil {
		return nil, fmt.Errorf("could not get support bundle config env vars: %w", err)
	}
	return result, nil
}

func CreateOrUpdateSupportBundleConfiguration(
	ctx context.Context,
	orgID uuid.UUID,
	envVars []types.SupportBundleConfigurationEnvVar,
) (*types.SupportBundleConfiguration, error) {
	var config types.SupportBundleConfiguration
	err := RunTxRR(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)

		rows, err := db.Query(
			ctx,
			`INSERT INTO SupportBundleConfiguration (organization_id)
			VALUES (@orgId)
			ON CONFLICT (organization_id) DO UPDATE SET organization_id = EXCLUDED.organization_id
			RETURNING id, created_at, organization_id`,
			pgx.NamedArgs{"orgId": orgID},
		)
		if err != nil {
			return fmt.Errorf("could not upsert support bundle configuration: %w", err)
		}
		result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SupportBundleConfiguration])
		if err != nil {
			return fmt.Errorf("could not upsert support bundle configuration: %w", err)
		}
		config = result

		if _, err := db.Exec(
			ctx,
			`DELETE FROM SupportBundleConfigurationEnvVar WHERE support_bundle_configuration_id = @configId`,
			pgx.NamedArgs{"configId": config.ID},
		); err != nil {
			return fmt.Errorf("could not delete existing env vars: %w", err)
		}

		if len(envVars) > 0 {
			_, err := db.CopyFrom(
				ctx,
				pgx.Identifier{"supportbundleconfigurationenvvar"},
				[]string{"support_bundle_configuration_id", "name", "redacted"},
				pgx.CopyFromSlice(len(envVars), func(i int) ([]any, error) {
					return []any{config.ID, envVars[i].Name, envVars[i].Redacted}, nil
				}),
			)
			if err != nil {
				return fmt.Errorf("could not insert env vars: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func DeleteSupportBundleConfiguration(ctx context.Context, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	result, err := db.Exec(
		ctx,
		`DELETE FROM SupportBundleConfiguration WHERE organization_id = @orgId`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		return fmt.Errorf("could not delete support bundle configuration: %w", err)
	}
	if result.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}

func ExistsSupportBundleConfiguration(ctx context.Context, orgID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	var exists bool
	err := db.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM SupportBundleConfiguration WHERE organization_id = @orgId)`,
		pgx.NamedArgs{"orgId": orgID},
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("could not check support bundle configuration existence: %w", err)
	}
	return exists, nil
}

// Bundles

const supportBundleWithDetailsOutputExpr = `
	sb.id,
	sb.created_at,
	sb.organization_id,
	sb.customer_organization_id,
	sb.created_by_user_account_id,
	sb.title,
	sb.description,
	sb.status,
	sb.access_token_id,
	u.name AS created_by_user_name,
	u.image_id AS created_by_image_id,
	co.name AS customer_organization_name,
	(SELECT count(*) FROM SupportBundleResource WHERE support_bundle_id = sb.id) AS resource_count
`

func GetSupportBundles(
	ctx context.Context, orgID uuid.UUID, customerOrgID *uuid.UUID,
) ([]types.SupportBundleWithDetails, error) {
	db := internalctx.GetDb(ctx)
	query := fmt.Sprintf(`
		SELECT %v
		FROM SupportBundle sb
		INNER JOIN UserAccount u ON sb.created_by_user_account_id = u.id
		INNER JOIN CustomerOrganization co ON sb.customer_organization_id = co.id
		WHERE sb.organization_id = @orgId`,
		supportBundleWithDetailsOutputExpr)

	args := pgx.NamedArgs{"orgId": orgID}
	if customerOrgID != nil {
		query += ` AND sb.customer_organization_id = @customerOrgId`
		args["customerOrgId"] = *customerOrgID
	}
	query += ` ORDER BY sb.created_at DESC`

	rows, err := db.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("could not query support bundles: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.SupportBundleWithDetails])
	if err != nil {
		return nil, fmt.Errorf("could not get support bundles: %w", err)
	}
	return result, nil
}

func GetSupportBundleByID(ctx context.Context, id, orgID uuid.UUID) (*types.SupportBundleWithDetails, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf(`
			SELECT %v
			FROM SupportBundle sb
			INNER JOIN UserAccount u ON sb.created_by_user_account_id = u.id
			INNER JOIN CustomerOrganization co ON sb.customer_organization_id = co.id
			WHERE sb.id = @id AND sb.organization_id = @orgId`,
			supportBundleWithDetailsOutputExpr),
		pgx.NamedArgs{"id": id, "orgId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query support bundle: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SupportBundleWithDetails])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not get support bundle: %w", err)
	}
	return &result, nil
}

func GetSupportBundleByIDAndAccessToken(
	ctx context.Context, id, accessTokenID uuid.UUID,
) (*types.SupportBundle, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT id, created_at, organization_id, customer_organization_id, created_by_user_account_id,
			title, description, status, access_token_id
		FROM SupportBundle
		WHERE id = @id AND access_token_id = @accessTokenId`,
		pgx.NamedArgs{"id": id, "accessTokenId": accessTokenID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query support bundle: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SupportBundle])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not get support bundle: %w", err)
	}
	return &result, nil
}

func CreateSupportBundle(ctx context.Context, bundle *types.SupportBundle) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO SupportBundle
			(organization_id, customer_organization_id, created_by_user_account_id,
			title, description, access_token_id)
		VALUES (@orgId, @customerOrgId, @userId, @title, @description, @accessTokenId)
		RETURNING id, created_at, organization_id, customer_organization_id, created_by_user_account_id,
			title, description, status, access_token_id`,
		pgx.NamedArgs{
			"orgId":         bundle.OrganizationID,
			"customerOrgId": bundle.CustomerOrganizationID,
			"userId":        bundle.CreatedByUserAccountID,
			"title":         bundle.Title,
			"description":   bundle.Description,
			"accessTokenId": bundle.AccessTokenID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create support bundle: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SupportBundle])
	if err != nil {
		return fmt.Errorf("could not create support bundle: %w", err)
	}
	*bundle = result
	return nil
}

func UpdateSupportBundleStatus(ctx context.Context, id, orgID uuid.UUID, status types.SupportBundleStatus) error {
	db := internalctx.GetDb(ctx)
	result, err := db.Exec(
		ctx,
		`UPDATE SupportBundle SET status = @status WHERE id = @id AND organization_id = @orgId`,
		pgx.NamedArgs{"id": id, "orgId": orgID, "status": status},
	)
	if err != nil {
		return fmt.Errorf("could not update support bundle status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}

func ClearSupportBundleAccessToken(ctx context.Context, bundleID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	if _, err := db.Exec(
		ctx,
		`UPDATE SupportBundle SET access_token_id = NULL WHERE id = @id`,
		pgx.NamedArgs{"id": bundleID},
	); err != nil {
		return fmt.Errorf("could not clear support bundle access token: %w", err)
	}
	return nil
}

// Resources

func GetSupportBundleResources(ctx context.Context, bundleID uuid.UUID) ([]types.SupportBundleResource, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT id, created_at, support_bundle_id, name, content
		FROM SupportBundleResource
		WHERE support_bundle_id = @bundleId
		ORDER BY created_at`,
		pgx.NamedArgs{"bundleId": bundleID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query support bundle resources: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.SupportBundleResource])
	if err != nil {
		return nil, fmt.Errorf("could not get support bundle resources: %w", err)
	}
	return result, nil
}

func CreateSupportBundleResource(ctx context.Context, resource *types.SupportBundleResource) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO SupportBundleResource (support_bundle_id, name, content)
		VALUES (@bundleId, @name, @content)
		RETURNING id, created_at, support_bundle_id, name, content`,
		pgx.NamedArgs{
			"bundleId": resource.SupportBundleID,
			"name":     resource.Name,
			"content":  resource.Content,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create support bundle resource: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SupportBundleResource])
	if err != nil {
		return fmt.Errorf("could not create support bundle resource: %w", err)
	}
	*resource = result
	return nil
}

// Comments

func GetSupportBundleComments(ctx context.Context, bundleID uuid.UUID) ([]types.SupportBundleCommentWithUser, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT c.id, c.created_at, c.support_bundle_id, c.user_account_id, c.content,
			u.name AS user_name, u.image_id AS user_image_id
		FROM SupportBundleComment c
		INNER JOIN UserAccount u ON c.user_account_id = u.id
		WHERE c.support_bundle_id = @bundleId
		ORDER BY c.created_at`,
		pgx.NamedArgs{"bundleId": bundleID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query support bundle comments: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.SupportBundleCommentWithUser])
	if err != nil {
		return nil, fmt.Errorf("could not get support bundle comments: %w", err)
	}
	return result, nil
}

func CreateSupportBundleComment(ctx context.Context, comment *types.SupportBundleComment) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO SupportBundleComment (support_bundle_id, user_account_id, content)
		VALUES (@bundleId, @userId, @content)
		RETURNING id, created_at, support_bundle_id, user_account_id, content`,
		pgx.NamedArgs{
			"bundleId": comment.SupportBundleID,
			"userId":   comment.UserAccountID,
			"content":  comment.Content,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create support bundle comment: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.SupportBundleComment])
	if err != nil {
		return fmt.Errorf("could not create support bundle comment: %w", err)
	}
	*comment = result
	return nil
}
