package db

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const (
	alertConfigurationOutputExpr = `
	c.id,
	c.created_at,
	c.organization_id,
	c.customer_organization_id,
	c.name,
	c.enabled,
	(
		SELECT array_agg(dt.id)
		FROM DeploymentTarget dt
		WHERE exists(
			SELECT 1 FROM AlertConfiguration_DeploymentTarget j
			WHERE j.alert_configuration_id = c.id AND j.deployment_target_id = dt.id
		)
	) AS deployment_target_ids,
	(
		SELECT array_agg(u.id)
		FROM UserAccount u
		WHERE exists(
			SELECT 1 FROM AlertConfiguration_Organization_UserAccount j
			WHERE j.alert_configuration_id = c.id
				AND j.user_account_id = u.id
		)
	) AS user_account_ids,
	(
		SELECT array_agg(row(` + userAccountOutputExpr + `))
		FROM UserAccount u
		WHERE exists(
			SELECT 1 FROM AlertConfiguration_Organization_UserAccount j
			WHERE j.alert_configuration_id = c.id
				AND j.user_account_id = u.id
		)
	) AS user_accounts,
	(
		SELECT array_agg(row(` + deploymentTargetOutputExprBase + `))
		FROM DeploymentTarget dt
		WHERE exists(
			SELECT 1 FROM AlertConfiguration_DeploymentTarget j
			WHERE j.alert_configuration_id = c.id AND j.deployment_target_id = dt.id
		)
	) AS deployment_targets
	`
)

func GetAlertConfigurationsForAllOrganizations(ctx context.Context) ([]types.AlertConfiguration, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT `+alertConfigurationOutputExpr+`
		FROM AlertConfiguration c
		WHERE c.enabled = true`,
	)
	if err != nil {
		return nil, err
	}

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AlertConfiguration])
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetAlertConfigurations(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]types.AlertConfiguration, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT `+alertConfigurationOutputExpr+`
		FROM AlertConfiguration c
		WHERE c.organization_id = @orgID
			AND ((@customerOrgIsNull AND customer_organization_id IS NULL) OR (customer_organization_id = @customerOrgID))`,
		pgx.NamedArgs{
			"orgID":             organizationID,
			"customerOrgID":     customerOrganizationID,
			"customerOrgIsNull": customerOrganizationID == nil,
		},
	)
	if err != nil {
		return nil, err
	}

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AlertConfiguration])
	if err != nil {
		return nil, err
	}

	return result, nil
}

func CountAlertConfigurations(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT count(id) FROM AlertConfiguration WHERE organization_id = @organizationID`,
		pgx.NamedArgs{"organizationID": organizationID},
	)
	if err != nil {
		return 0, err
	}

	return pgx.CollectExactlyOneRow(rows, pgx.RowTo[int64])
}

func GetAlertConfigurationsForDeploymentTarget(
	ctx context.Context,
	deploymentTargetID uuid.UUID,
) ([]types.AlertConfiguration, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT `+alertConfigurationOutputExpr+`
		FROM AlertConfiguration c
		WHERE exists(
			SELECT 1 FROM AlertConfiguration_DeploymentTarget j
			WHERE j.alert_configuration_id = c.id
				AND j.deployment_target_id = @deploymentTargetID
		)`,
		pgx.NamedArgs{
			"deploymentTargetID": deploymentTargetID,
		},
	)
	if err != nil {
		return nil, err
	}

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.AlertConfiguration])
	if err != nil {
		return nil, err
	}

	return result, nil
}

func CreateAlertConfiguration(ctx context.Context, config *types.AlertConfiguration) error {
	return RunTxRR(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)

		rows, err := db.Query(
			ctx,
			`WITH inserted AS (
			INSERT INTO AlertConfiguration (
				organization_id,
				customer_organization_id,
				name,
				enabled
			) VALUES (
				@organizationID,
				@customerOrganizationID,
				@name,
				@enabled
			)
			RETURNING id
		)
		SELECT id FROM inserted`,
			pgx.NamedArgs{
				"organizationID":         config.OrganizationID,
				"customerOrganizationID": config.CustomerOrganizationID,
				"name":                   config.Name,
				"enabled":                config.Enabled,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to insert AlertConfiguration: %w", err)
		}

		if insertedID, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[uuid.UUID]); err != nil {
			return fmt.Errorf("failed to collect inserted ID: %w", err)
		} else {
			config.ID = insertedID
		}

		if err := updateAlertConfigUserAccountIDs(ctx, config); err != nil {
			return fmt.Errorf("failed to update user account IDs: %w", err)
		}

		if err := updateAlertConfigDeploymentTargetIDs(ctx, config); err != nil {
			return fmt.Errorf("failed to update deployment target IDs: %w", err)
		}

		return getAlertConfigurationInto(ctx, config.ID, config)
	})
}

func UpdateAlertConfiguration(ctx context.Context, config *types.AlertConfiguration) error {
	return RunTxRR(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)

		_, err := db.Exec(
			ctx,
			`UPDATE AlertConfiguration SET
				name = @name,
				enabled = @enabled
			WHERE id = @id
				AND organization_id = @orgID
				AND ((@customerOrgIsNull AND customer_organization_id IS NULL) OR (customer_organization_id = @customerOrgID))`,
			pgx.NamedArgs{
				"id":                config.ID,
				"name":              config.Name,
				"enabled":           config.Enabled,
				"orgID":             config.OrganizationID,
				"customerOrgID":     config.CustomerOrganizationID,
				"customerOrgIsNull": config.CustomerOrganizationID == nil,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to update AlertConfiguration: %w", err)
		}

		if err := updateAlertConfigUserAccountIDs(ctx, config); err != nil {
			return fmt.Errorf("failed to update user account IDs: %w", err)
		}

		if err := updateAlertConfigDeploymentTargetIDs(ctx, config); err != nil {
			return fmt.Errorf("failed to update deployment target IDs: %w", err)
		}

		return getAlertConfigurationInto(ctx, config.ID, config)
	})
}

func updateAlertConfigUserAccountIDs(ctx context.Context, config *types.AlertConfiguration) error {
	db := internalctx.GetDb(ctx)

	cmd, err := db.Exec(
		ctx,
		`INSERT INTO AlertConfiguration_Organization_UserAccount (
			alert_configuration_id,
			organization_id,
			user_account_id
		)
		(SELECT @id, @organizationID, id FROM UserAccount WHERE id = any(@userAccountIDs))
		ON CONFLICT (alert_configuration_id, organization_id, user_account_id) DO NOTHING`,
		pgx.NamedArgs{
			"id":             config.ID,
			"organizationID": config.OrganizationID,
			"userAccountIDs": config.UserAccountIDs,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to insert user account IDs: %w", err)
	}

	log := internalctx.GetLogger(ctx)
	log.Debug("inserted config user relations", zap.Int64("rowsAffected", cmd.RowsAffected()))

	_, err = db.Exec(
		ctx,
		`DELETE FROM AlertConfiguration_Organization_UserAccount
		WHERE alert_configuration_id = @id
			AND NOT user_account_id = any(@userAccountIDs)`,
		pgx.NamedArgs{
			"id":             config.ID,
			"userAccountIDs": config.UserAccountIDs,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to delete user account IDs: %w", err)
	}

	log.Debug("deleted config user relations", zap.Int64("rowsAffected", cmd.RowsAffected()))

	return nil
}

func updateAlertConfigDeploymentTargetIDs(ctx context.Context, config *types.AlertConfiguration) error {
	db := internalctx.GetDb(ctx)

	cmd, err := db.Exec(
		ctx,
		`INSERT INTO AlertConfiguration_DeploymentTarget (
			alert_configuration_id,
			deployment_target_id
		)
		SELECT @id, id FROM DeploymentTarget WHERE id = any(@deploymentTargetIDs)
		ON CONFLICT (alert_configuration_id, deployment_target_id) DO NOTHING`,
		pgx.NamedArgs{
			"id":                  config.ID,
			"deploymentTargetIDs": config.DeploymentTargetIDs,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to insert deployment target IDs: %w", err)
	}

	log := internalctx.GetLogger(ctx)
	log.Debug("inserted config DeploymentTarget relations", zap.Int64("rowsAffected", cmd.RowsAffected()))

	_, err = db.Exec(
		ctx,
		`DELETE FROM AlertConfiguration_DeploymentTarget
		WHERE alert_configuration_id = @id
			AND NOT deployment_target_id = any(@deploymentTargetIDs)`,
		pgx.NamedArgs{
			"id":                  config.ID,
			"deploymentTargetIDs": config.DeploymentTargetIDs,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to delete deployment target IDs: %w", err)
	}

	log.Debug("deleted config DeploymentTarget relations", zap.Int64("rowsAffected", cmd.RowsAffected()))

	return nil
}

func getAlertConfigurationInto(ctx context.Context, id uuid.UUID, target *types.AlertConfiguration) error {
	db := internalctx.GetDb(ctx)
	if rows, err := db.Query(
		ctx,
		`SELECT`+alertConfigurationOutputExpr+
			`FROM AlertConfiguration c WHERE c.id = @id`,
		pgx.NamedArgs{"id": id},
	); err != nil {
		return fmt.Errorf("failed to query AlertConfiguration: %w", err)
	} else if result, err := pgx.CollectExactlyOneRow(
		rows,
		pgx.RowToStructByName[types.AlertConfiguration],
	); err != nil {
		return fmt.Errorf("failed to collect AlertConfiguration: %w", err)
	} else {
		*target = result
		return nil
	}
}

func DeleteAlertConfiguration(
	ctx context.Context,
	id uuid.UUID,
	organizationID uuid.UUID,
	customerOrgID *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)

	cmd, err := db.Exec(
		ctx,
		`DELETE FROM AlertConfiguration
		WHERE id = @id
			AND organization_id = @organizationID
			AND ((@customerOrgIsNull AND customer_organization_id IS NULL) OR (customer_organization_id = @customerOrgID))`,
		pgx.NamedArgs{
			"id":                id,
			"organizationID":    organizationID,
			"customerOrgIsNull": customerOrgID == nil,
			"customerOrgID":     customerOrgID,
		},
	)

	if err == nil && cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("failed to delete AlertConfiguration: %w", err)
	}

	return nil
}

func DeleteAlertConfigurationsWithOrganizationID(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	db := internalctx.GetDb(ctx)

	cmd, err := db.Exec(
		ctx,
		`DELETE FROM AlertConfiguration WHERE organization_id = @organizationID`,
		pgx.NamedArgs{"organizationID": organizationID},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete AlertConfiguration: %w", err)
	}

	return cmd.RowsAffected(), nil
}
