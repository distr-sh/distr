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

const (
	applicationLicenseOutputExpr = `
		al.id, al.created_at, al.name, al.expires_at, al.application_id, al.organization_id,
		al.customer_organization_id, al.registry_url, al.registry_username, al.registry_password
	`
	applicationLicenseWithVersionsOutputExpr = applicationLicenseOutputExpr + `,
		coalesce((
		   	SELECT array_agg(
				row(av.id, av.created_at, av.archived_at, av.name, av.application_id,
					av.chart_type, av.chart_name, av.chart_url, av.chart_version)
				ORDER BY av.created_at ASC
			)
		   	FROM ApplicationLicense_ApplicationVersion alav
				LEFT JOIN applicationversion av ON alav.application_version_id = av.id
		   	WHERE alav.application_license_id = al.id
		   ), array[]::record[]
		) as versions
	`
	applicationLicenseCompleteOutputExpr = applicationLicenseWithVersionsOutputExpr + `,
		(a.id, a.created_at, a.organization_id, a.name, a.type) as application,
		CASE WHEN al.customer_organization_id IS NOT NULL
			THEN (` + customerOrganizationOutputExpr + `)
		END as customer_organization
	`
)

func CreateApplicationLicense(ctx context.Context, license *types.ApplicationLicenseBase) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO ApplicationLicense AS al (
			name, expires_at, application_id, organization_id, customer_organization_id, registry_url, registry_username,
			registry_password
		) VALUES (
			@name, @expiresAt, @applicationId, @organizationId, @customerOrganizationId, @registryUrl, @registryUsername,
			@registryPassword
		) RETURNING`+applicationLicenseOutputExpr,
		pgx.NamedArgs{
			"name":                   license.Name,
			"expiresAt":              license.ExpiresAt,
			"applicationId":          license.ApplicationID,
			"organizationId":         license.OrganizationID,
			"customerOrganizationId": license.CustomerOrganizationID,
			"registryUrl":            license.RegistryURL,
			"registryUsername":       license.RegistryUsername,
			"registryPassword":       license.RegistryPassword,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert ApplicationLicense: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ApplicationLicenseBase]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*license = result
		return nil
	}
}

func UpdateApplicationLicense(ctx context.Context, license *types.ApplicationLicenseBase) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`UPDATE ApplicationLicense AS al SET
			name = @name,
            expires_at = @expiresAt,
            customer_organization_id = @customerOrganizationId,
            registry_url = @registryUrl,
            registry_username = @registryUsername,
            registry_password = @registryPassword
		 WHERE al.id = @id RETURNING`+applicationLicenseOutputExpr,
		pgx.NamedArgs{
			"id":                     license.ID,
			"name":                   license.Name,
			"expiresAt":              license.ExpiresAt,
			"customerOrganizationId": license.CustomerOrganizationID,
			"registryUrl":            license.RegistryURL,
			"registryUsername":       license.RegistryUsername,
			"registryPassword":       license.RegistryPassword,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert ApplicationLicense: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ApplicationLicenseBase]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*license = result
		return nil
	}
}

func RevokeApplicationLicenseWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, "UPDATE ApplicationLicense SET expires_at = now() WHERE id = @id", pgx.NamedArgs{"id": id})
	if err == nil && cmd.RowsAffected() < 1 {
		err = apierrors.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("could not update ApplicationLicense: %w", err)
	} else {
		return nil
	}
}

func AddVersionToApplicationLicense(
	ctx context.Context,
	license *types.ApplicationLicenseBase,
	id uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		`INSERT INTO ApplicationLicense_ApplicationVersion (application_version_id, application_license_id)
		VALUES (@applicationVersionId, @applicationLicenseId)
		ON CONFLICT (application_version_id, application_license_id) DO NOTHING`,
		pgx.NamedArgs{
			"applicationVersionId": id,
			"applicationLicenseId": license.ID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert relation: %w", err)
	}
	return nil
}

func RemoveVersionFromApplicationLicense(
	ctx context.Context,
	license *types.ApplicationLicenseBase,
	id uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		`DELETE FROM ApplicationLicense_ApplicationVersion
		WHERE application_license_id = @applicationLicenseId
		AND application_version_id = @applicationVersionId`,
		pgx.NamedArgs{
			"applicationLicenseId": license.ID,
			"applicationVersionId": id,
		},
	)
	if err != nil {
		return fmt.Errorf("could not delete relation: %w", err)
	} else {
		return nil
	}
}

func GetApplicationLicensesWithOrganizationID(
	ctx context.Context,
	organizationID uuid.UUID,
	applicationID *uuid.UUID,
) ([]types.ApplicationLicense, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		"SELECT "+applicationLicenseCompleteOutputExpr+
			"FROM ApplicationLicense al "+
			"LEFT JOIN Application a ON al.application_id = a.id "+
			"LEFT JOIN CustomerOrganization co ON al.customer_organization_id = co.id "+
			"WHERE al.organization_id = @organizationId "+
			andApplicationIdMatchesOrEmpty(applicationID),
		pgx.NamedArgs{
			"organizationId": organizationID,
			"applicationId":  applicationID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query ApplicationLicense: %w", err)
	}

	if result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ApplicationLicense]); err != nil {
		return nil, fmt.Errorf("could not collect ApplicationLicense: %w", err)
	} else {
		return result, nil
	}
}

func GetApplicationLicensesWithCustomerOrganizationID(
	ctx context.Context,
	customerOrganizationID, organizationID uuid.UUID,
	applicationID *uuid.UUID,
) ([]types.ApplicationLicense, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		"SELECT "+applicationLicenseCompleteOutputExpr+
			"FROM ApplicationLicense al "+
			"LEFT JOIN Application a ON al.application_id = a.id "+
			"LEFT JOIN CustomerOrganization co ON al.customer_organization_id = co.id "+
			"WHERE al.customer_organization_id = @customerOrganizationId AND al.organization_id = @organizationId "+
			andApplicationIdMatchesOrEmpty(applicationID),
		pgx.NamedArgs{
			"customerOrganizationId": customerOrganizationID,
			"organizationId":         organizationID,
			"applicationId":          applicationID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query ApplicationLicense: %w", err)
	}

	if result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ApplicationLicense]); err != nil {
		return nil, fmt.Errorf("could not collect ApplicationLicense: %w", err)
	} else {
		return result, nil
	}
}

func andApplicationIdMatchesOrEmpty(applicationID *uuid.UUID) string {
	if applicationID != nil {
		return " AND al.application_id = @applicationId "
	}
	return ""
}

func GetApplicationLicenseByID(ctx context.Context, id uuid.UUID) (*types.ApplicationLicense, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		"SELECT "+applicationLicenseCompleteOutputExpr+
			"FROM ApplicationLicense al "+
			"LEFT JOIN Application a ON al.application_id = a.id "+
			"LEFT JOIN CustomerOrganization co ON al.customer_organization_id = co.id "+
			"WHERE al.id = @id ",
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query ApplicationLicense: %w", err)
	}

	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ApplicationLicense]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not collect ApplicationLicense: %w", err)
	} else {
		return &result, nil
	}
}

func SetApplicationLicenseVersions(ctx context.Context, licenseID uuid.UUID, versionIDs []uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(
		ctx,
		`INSERT INTO ApplicationLicense_ApplicationVersion (application_license_id, application_version_id)
		SELECT @licenseId, id FROM ApplicationVersion WHERE id = any(@versionIds)
		ON CONFLICT (application_license_id, application_version_id) DO NOTHING`,
		pgx.NamedArgs{
			"licenseId":  licenseID,
			"versionIds": versionIDs,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert version relations: %w", err)
	}
	_, err = db.Exec(
		ctx,
		`DELETE FROM ApplicationLicense_ApplicationVersion
		WHERE application_license_id = @licenseId
			AND NOT (application_version_id = any(@versionIds))`,
		pgx.NamedArgs{
			"licenseId":  licenseID,
			"versionIds": versionIDs,
		},
	)
	if err != nil {
		return fmt.Errorf("could not delete version relations: %w", err)
	}
	return nil
}

func GetDeploymentsUsingVersionsNotInList(
	ctx context.Context,
	licenseID uuid.UUID,
	allowedVersionIDs []uuid.UUID,
) ([]types.DeploymentVersionUsage, error) {
	if len(allowedVersionIDs) == 0 {
		return nil, nil
	}
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT
			d.id AS deployment_id,
			dt.name AS deployment_target_name,
			dr.application_version_id AS application_version_id,
			av.name AS application_version_name
		FROM Deployment d
			JOIN DeploymentTarget dt ON d.deployment_target_id = dt.id
			JOIN (
				SELECT deployment_id, max(created_at) AS max_created_at
				FROM DeploymentRevision
				GROUP BY deployment_id
			) dr_max ON d.id = dr_max.deployment_id
			JOIN DeploymentRevision dr
				ON d.id = dr.deployment_id
				AND dr.created_at = dr_max.max_created_at
			JOIN ApplicationVersion av ON dr.application_version_id = av.id
		WHERE d.application_license_id = @licenseId
			AND dr.application_version_id != ALL(@allowedVersionIds)`,
		pgx.NamedArgs{
			"licenseId":         licenseID,
			"allowedVersionIds": allowedVersionIDs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query deployments using unlicensed versions: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentVersionUsage])
	if err != nil {
		return nil, fmt.Errorf("could not collect deployment version usage: %w", err)
	}
	return result, nil
}

func DeleteApplicationLicenseWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `DELETE FROM ApplicationLicense WHERE id = @id`, pgx.NamedArgs{"id": id})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else if cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("could not delete ApplicationLicense: %w", err)
	}

	return nil
}

func DeleteApplicationLicensesWithOrganizationID(ctx context.Context, orgID uuid.UUID) (int64, error) {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM ApplicationLicense WHERE organization_id = @orgId`,
		pgx.NamedArgs{"orgId": orgID},
	)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return 0, fmt.Errorf("could not delete ApplicationLicenses: %w", err)
	}

	return cmd.RowsAffected(), nil
}

func DeleteApplicationLicensesWithOrganizationSubscriptionType(
	ctx context.Context,
	subscriptionType []types.SubscriptionType,
) (int64, error) {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM ApplicationLicense WHERE organization_id IN (
			SELECT id FROM Organization WHERE subscription_type = ANY(@subscriptionType)
		)`,
		pgx.NamedArgs{"subscriptionType": subscriptionType},
	)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return 0, fmt.Errorf("could not delete ApplicationLicenses: %w", err)
	}

	return cmd.RowsAffected(), nil
}
