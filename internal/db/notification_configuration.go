package db

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var updateNotificationConfigurationOutputExpr = `
	c.id,
	c.created_at,
	c.organization_id,
	c.customer_organization_id,
	c.name,
	c.enabled,
	(
		SELECT array_agg(row(a.id, a.name, a.type, a.image_id) ORDER BY a.name)
		FROM Application a
		WHERE EXISTS (
			SELECT 1 FROM UpdateNotificationConfiguration_Application j
			WHERE j.update_notification_configuration_id = c.id AND j.application_id = a.id
		)
	) AS applications,
	(
		SELECT array_agg(row(a.id, a.name, a.image_id) ORDER BY a.name)
		FROM Artifact a
		WHERE EXISTS (
			SELECT 1 FROM UpdateNotificationConfiguration_Artifact j
			WHERE j.update_notification_configuration_id = c.id AND j.artifact_id = a.id
		)
	) AS artifacts,
	-- The recipients are joined through Organization_UserAccount so that the membership deciding
	-- what a recipient may see travels with the user account.
	(
		SELECT array_agg(row(u.id, u.email, u.name, oua.customer_organization_id, oua.partner_organization_id))
		FROM UserAccount u
		JOIN Organization_UserAccount oua ON oua.user_account_id = u.id AND oua.organization_id = c.organization_id
		WHERE EXISTS (
			SELECT 1 FROM UpdateNotificationConfiguration_Organization_UserAccount j
			WHERE j.update_notification_configuration_id = c.id AND j.user_account_id = u.id
		)
	) AS recipients`

func GetUpdateNotificationConfigurations(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]types.UpdateNotificationConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+updateNotificationConfigurationOutputExpr+`
		FROM UpdateNotificationConfiguration c
		WHERE c.organization_id = @organizationID
			AND ((@customerOrgIsNull AND c.customer_organization_id IS NULL)
				OR c.customer_organization_id = @customerOrganizationID)
		ORDER BY c.name`,
		pgx.NamedArgs{
			"organizationID":         organizationID,
			"customerOrganizationID": customerOrganizationID,
			"customerOrgIsNull":      customerOrganizationID == nil,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query UpdateNotificationConfiguration: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.UpdateNotificationConfiguration])
}

// GetUpdateNotificationConfigurationsForApplication returns the enabled configurations that watch
// the given application.
func GetUpdateNotificationConfigurationsForApplication(
	ctx context.Context,
	applicationID uuid.UUID,
) ([]types.UpdateNotificationConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+updateNotificationConfigurationOutputExpr+`
		FROM UpdateNotificationConfiguration c
		WHERE c.enabled
			AND EXISTS (
				SELECT 1 FROM UpdateNotificationConfiguration_Application j
				WHERE j.update_notification_configuration_id = c.id
					AND j.application_id = @applicationID
			)`,
		pgx.NamedArgs{"applicationID": applicationID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query UpdateNotificationConfiguration: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.UpdateNotificationConfiguration])
}

// GetUpdateNotificationConfigurationsForArtifact returns the enabled configurations that watch the
// given artifact. An artifact that mirrors an upstream registry never matches: its versions all
// appear at once when the sync runs, which would mean one mail per upstream tag.
func GetUpdateNotificationConfigurationsForArtifact(
	ctx context.Context,
	artifactID uuid.UUID,
) ([]types.UpdateNotificationConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+updateNotificationConfigurationOutputExpr+`
		FROM UpdateNotificationConfiguration c
		WHERE c.enabled
			AND EXISTS (
				SELECT 1 FROM UpdateNotificationConfiguration_Artifact j
				JOIN Artifact a ON a.id = j.artifact_id
				WHERE j.update_notification_configuration_id = c.id
					AND j.artifact_id = @artifactID
					AND a.upstream_url IS NULL
			)`,
		pgx.NamedArgs{"artifactID": artifactID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query UpdateNotificationConfiguration: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.UpdateNotificationConfiguration])
}

func CreateUpdateNotificationConfiguration(
	ctx context.Context,
	config *types.UpdateNotificationConfiguration,
) error {
	return RunTxRR(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)
		rows, err := db.Query(
			ctx,
			`INSERT INTO UpdateNotificationConfiguration (
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
			RETURNING id`,
			updateNotificationConfigurationArgs(config),
		)
		if err != nil {
			return fmt.Errorf("failed to insert UpdateNotificationConfiguration: %w", err)
		}
		if id, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[uuid.UUID]); err != nil {
			return fmt.Errorf("failed to collect inserted ID: %w", err)
		} else {
			config.ID = id
		}
		return updateUpdateNotificationConfigurationLinks(ctx, config)
	})
}

func UpdateUpdateNotificationConfiguration(
	ctx context.Context,
	config *types.UpdateNotificationConfiguration,
) error {
	return RunTxRR(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)
		cmd, err := db.Exec(
			ctx,
			`UPDATE UpdateNotificationConfiguration SET
				name = @name,
				enabled = @enabled
			WHERE id = @id
				AND organization_id = @organizationID
				AND ((@customerOrgIsNull AND customer_organization_id IS NULL)
					OR customer_organization_id = @customerOrganizationID)`,
			updateNotificationConfigurationArgs(config),
		)
		if err != nil {
			return fmt.Errorf("failed to update UpdateNotificationConfiguration: %w", err)
		} else if cmd.RowsAffected() == 0 {
			return apierrors.ErrNotFound
		}
		return updateUpdateNotificationConfigurationLinks(ctx, config)
	})
}

func DeleteUpdateNotificationConfiguration(
	ctx context.Context,
	id, organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		`DELETE FROM UpdateNotificationConfiguration
		WHERE id = @id
			AND organization_id = @organizationID
			AND ((@customerOrgIsNull AND customer_organization_id IS NULL)
				OR customer_organization_id = @customerOrganizationID)`,
		pgx.NamedArgs{
			"id":                     id,
			"organizationID":         organizationID,
			"customerOrganizationID": customerOrganizationID,
			"customerOrgIsNull":      customerOrganizationID == nil,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to delete UpdateNotificationConfiguration: %w", err)
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}

func updateNotificationConfigurationArgs(config *types.UpdateNotificationConfiguration) pgx.NamedArgs {
	return pgx.NamedArgs{
		"id":                     config.ID,
		"organizationID":         config.OrganizationID,
		"customerOrganizationID": config.CustomerOrganizationID,
		"customerOrgIsNull":      config.CustomerOrganizationID == nil,
		"name":                   config.Name,
		"enabled":                config.Enabled,
	}
}

func updateUpdateNotificationConfigurationLinks(
	ctx context.Context,
	config *types.UpdateNotificationConfiguration,
) error {
	if err := replaceNotificationLinks(ctx, notificationLink{
		table:        "UpdateNotificationConfiguration_Application",
		configColumn: "update_notification_configuration_id",
		valueColumn:  "application_id",
		valueTable:   "Application",
		customerCondition: `EXISTS (
			SELECT 1 FROM ApplicationEntitlement ae
			WHERE ae.application_id = v.id
				AND ae.customer_organization_id = @customerOrganizationID
				AND (ae.expires_at IS NULL OR ae.expires_at > now())
		)`,
	}, config.ID, config.OrganizationID, config.CustomerOrganizationID, config.ApplicationIDs); err != nil {
		return err
	}
	if err := replaceNotificationLinks(ctx, notificationLink{
		table:        "UpdateNotificationConfiguration_Artifact",
		configColumn: "update_notification_configuration_id",
		valueColumn:  "artifact_id",
		valueTable:   "Artifact",
		// Every version of a mirrored artifact appears at once when the upstream sync runs, so
		// announcing them would mean a mail per upstream tag.
		valueCondition: "v.upstream_url IS NULL",
		customerCondition: `EXISTS (
			SELECT 1 FROM ArtifactEntitlement ae
			JOIN ArtifactEntitlement_Artifact aea ON aea.artifact_entitlement_id = ae.id
			WHERE aea.artifact_id = v.id
				AND ae.customer_organization_id = @customerOrganizationID
				AND (ae.expires_at IS NULL OR ae.expires_at > now())
		)`,
	}, config.ID, config.OrganizationID, config.CustomerOrganizationID, config.ArtifactIDs); err != nil {
		return err
	}
	if err := replaceNotificationRecipients(
		ctx,
		"UpdateNotificationConfiguration_Organization_UserAccount",
		"update_notification_configuration_id",
		config.ID,
		config.OrganizationID,
		config.CustomerOrganizationID,
		config.UserAccountIDs,
	); err != nil {
		return err
	}
	return getUpdateNotificationConfigurationInto(ctx, config.ID, config)
}

func getUpdateNotificationConfigurationInto(
	ctx context.Context,
	id uuid.UUID,
	target *types.UpdateNotificationConfiguration,
) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+updateNotificationConfigurationOutputExpr+`
		FROM UpdateNotificationConfiguration c WHERE c.id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return fmt.Errorf("failed to query UpdateNotificationConfiguration: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.UpdateNotificationConfiguration])
	if err != nil {
		return fmt.Errorf("failed to collect UpdateNotificationConfiguration: %w", err)
	}
	*target = result
	return nil
}
