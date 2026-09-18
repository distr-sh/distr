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

// notificationRecipientsExpr collects the recipients of a configuration. It joins through
// Organization_UserAccount so that the membership deciding what a recipient may see travels with
// the user account.
func notificationRecipientsExpr(linkTable, configColumn string) string {
	return `(
		SELECT array_agg(row(u.id, u.email, u.name, oua.customer_organization_id, oua.partner_organization_id))
		FROM UserAccount u
		JOIN Organization_UserAccount oua ON oua.user_account_id = u.id AND oua.organization_id = c.organization_id
		WHERE EXISTS (
			SELECT 1 FROM ` + linkTable + ` j
			WHERE j.` + configColumn + ` = c.id AND j.user_account_id = u.id
		)
	) AS recipients`
}

var applicationNotificationConfigurationOutputExpr = `
	c.id,
	c.created_at,
	c.organization_id,
	c.customer_organization_id,
	c.name,
	c.enabled,
	c.update_available_trigger_enabled,
	c.customer_message,
	(
		SELECT array_agg(row(a.id, a.name, a.type, a.image_id) ORDER BY a.name)
		FROM Application a
		WHERE EXISTS (
			SELECT 1 FROM ApplicationNotificationConfiguration_Application j
			WHERE j.application_notification_configuration_id = c.id AND j.application_id = a.id
		)
	) AS applications,
	` + notificationRecipientsExpr(
	"ApplicationNotificationConfiguration_Organization_UserAccount",
	"application_notification_configuration_id",
)

var artifactNotificationConfigurationOutputExpr = `
	c.id,
	c.created_at,
	c.organization_id,
	c.customer_organization_id,
	c.name,
	c.enabled,
	c.new_version_trigger_enabled,
	c.customer_message,
	(
		SELECT array_agg(row(a.id, a.name, a.image_id) ORDER BY a.name)
		FROM Artifact a
		WHERE EXISTS (
			SELECT 1 FROM ArtifactNotificationConfiguration_Artifact j
			WHERE j.artifact_notification_configuration_id = c.id AND j.artifact_id = a.id
		)
	) AS artifacts,
	` + notificationRecipientsExpr(
	"ArtifactNotificationConfiguration_Organization_UserAccount",
	"artifact_notification_configuration_id",
)

func GetApplicationNotificationConfigurations(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]types.ApplicationNotificationConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+applicationNotificationConfigurationOutputExpr+`
		FROM ApplicationNotificationConfiguration c
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
		return nil, fmt.Errorf("failed to query ApplicationNotificationConfiguration: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.ApplicationNotificationConfiguration])
}

// GetApplicationNotificationConfigurationsForApplication returns the enabled configurations that
// have the update available trigger switched on for the given application.
func GetApplicationNotificationConfigurationsForApplication(
	ctx context.Context,
	applicationID uuid.UUID,
) ([]types.ApplicationNotificationConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+applicationNotificationConfigurationOutputExpr+`
		FROM ApplicationNotificationConfiguration c
		WHERE c.enabled
			AND c.update_available_trigger_enabled
			AND EXISTS (
				SELECT 1 FROM ApplicationNotificationConfiguration_Application j
				WHERE j.application_notification_configuration_id = c.id
					AND j.application_id = @applicationID
			)`,
		pgx.NamedArgs{"applicationID": applicationID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query ApplicationNotificationConfiguration: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.ApplicationNotificationConfiguration])
}

func CreateApplicationNotificationConfiguration(
	ctx context.Context,
	config *types.ApplicationNotificationConfiguration,
) error {
	return RunTxRR(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)
		rows, err := db.Query(
			ctx,
			`INSERT INTO ApplicationNotificationConfiguration (
				organization_id,
				customer_organization_id,
				name,
				enabled,
				update_available_trigger_enabled,
				customer_message
			) VALUES (
				@organizationID,
				@customerOrganizationID,
				@name,
				@enabled,
				@updateAvailableTriggerEnabled,
				@customerMessage
			)
			RETURNING id`,
			applicationNotificationConfigurationArgs(config),
		)
		if err != nil {
			return fmt.Errorf("failed to insert ApplicationNotificationConfiguration: %w", err)
		}
		if id, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[uuid.UUID]); err != nil {
			return fmt.Errorf("failed to collect inserted ID: %w", err)
		} else {
			config.ID = id
		}
		return updateApplicationNotificationConfigurationLinks(ctx, config)
	})
}

func UpdateApplicationNotificationConfiguration(
	ctx context.Context,
	config *types.ApplicationNotificationConfiguration,
) error {
	return RunTxRR(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)
		cmd, err := db.Exec(
			ctx,
			`UPDATE ApplicationNotificationConfiguration SET
				name = @name,
				enabled = @enabled,
				update_available_trigger_enabled = @updateAvailableTriggerEnabled,
				customer_message = @customerMessage
			WHERE id = @id
				AND organization_id = @organizationID
				AND ((@customerOrgIsNull AND customer_organization_id IS NULL)
					OR customer_organization_id = @customerOrganizationID)`,
			applicationNotificationConfigurationArgs(config),
		)
		if err != nil {
			return fmt.Errorf("failed to update ApplicationNotificationConfiguration: %w", err)
		} else if cmd.RowsAffected() == 0 {
			return apierrors.ErrNotFound
		}
		return updateApplicationNotificationConfigurationLinks(ctx, config)
	})
}

func applicationNotificationConfigurationArgs(
	config *types.ApplicationNotificationConfiguration,
) pgx.NamedArgs {
	return pgx.NamedArgs{
		"id":                            config.ID,
		"organizationID":                config.OrganizationID,
		"customerOrganizationID":        config.CustomerOrganizationID,
		"customerOrgIsNull":             config.CustomerOrganizationID == nil,
		"name":                          config.Name,
		"enabled":                       config.Enabled,
		"updateAvailableTriggerEnabled": config.UpdateAvailableTriggerEnabled,
		"customerMessage":               config.CustomerMessage,
	}
}

func updateApplicationNotificationConfigurationLinks(
	ctx context.Context,
	config *types.ApplicationNotificationConfiguration,
) error {
	if err := replaceNotificationLinks(ctx, notificationLink{
		table:        "ApplicationNotificationConfiguration_Application",
		configColumn: "application_notification_configuration_id",
		valueColumn:  "application_id",
		valueTable:   "Application",
	}, config.ID, config.OrganizationID, config.ApplicationIDs); err != nil {
		return err
	}
	if err := replaceNotificationRecipients(
		ctx,
		"ApplicationNotificationConfiguration_Organization_UserAccount",
		"application_notification_configuration_id",
		config.ID,
		config.OrganizationID,
		config.UserAccountIDs,
	); err != nil {
		return err
	}
	return getApplicationNotificationConfigurationInto(ctx, config.ID, config)
}

func getApplicationNotificationConfigurationInto(
	ctx context.Context,
	id uuid.UUID,
	target *types.ApplicationNotificationConfiguration,
) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+applicationNotificationConfigurationOutputExpr+`
		FROM ApplicationNotificationConfiguration c WHERE c.id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return fmt.Errorf("failed to query ApplicationNotificationConfiguration: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(
		rows,
		pgx.RowToStructByName[types.ApplicationNotificationConfiguration],
	)
	if err != nil {
		return fmt.Errorf("failed to collect ApplicationNotificationConfiguration: %w", err)
	}
	*target = result
	return nil
}

func DeleteApplicationNotificationConfiguration(
	ctx context.Context,
	id, organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) error {
	return deleteNotificationConfiguration(
		ctx, "ApplicationNotificationConfiguration", id, organizationID, customerOrganizationID,
	)
}

func GetArtifactNotificationConfigurations(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]types.ArtifactNotificationConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+artifactNotificationConfigurationOutputExpr+`
		FROM ArtifactNotificationConfiguration c
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
		return nil, fmt.Errorf("failed to query ArtifactNotificationConfiguration: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.ArtifactNotificationConfiguration])
}

// GetArtifactNotificationConfigurationsForArtifact returns the enabled configurations that have
// the new version trigger switched on for the given artifact. An artifact that mirrors an upstream
// registry never matches: its versions all appear at once when the sync runs, which would mean one
// mail per upstream tag.
func GetArtifactNotificationConfigurationsForArtifact(
	ctx context.Context,
	artifactID uuid.UUID,
) ([]types.ArtifactNotificationConfiguration, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+artifactNotificationConfigurationOutputExpr+`
		FROM ArtifactNotificationConfiguration c
		WHERE c.enabled
			AND c.new_version_trigger_enabled
			AND EXISTS (
				SELECT 1 FROM ArtifactNotificationConfiguration_Artifact j
				JOIN Artifact a ON a.id = j.artifact_id
				WHERE j.artifact_notification_configuration_id = c.id
					AND j.artifact_id = @artifactID
					AND a.upstream_url IS NULL
			)`,
		pgx.NamedArgs{"artifactID": artifactID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query ArtifactNotificationConfiguration: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[types.ArtifactNotificationConfiguration])
}

func CreateArtifactNotificationConfiguration(
	ctx context.Context,
	config *types.ArtifactNotificationConfiguration,
) error {
	return RunTxRR(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)
		rows, err := db.Query(
			ctx,
			`INSERT INTO ArtifactNotificationConfiguration (
				organization_id,
				customer_organization_id,
				name,
				enabled,
				new_version_trigger_enabled,
				customer_message
			) VALUES (
				@organizationID,
				@customerOrganizationID,
				@name,
				@enabled,
				@newVersionTriggerEnabled,
				@customerMessage
			)
			RETURNING id`,
			artifactNotificationConfigurationArgs(config),
		)
		if err != nil {
			return fmt.Errorf("failed to insert ArtifactNotificationConfiguration: %w", err)
		}
		if id, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[uuid.UUID]); err != nil {
			return fmt.Errorf("failed to collect inserted ID: %w", err)
		} else {
			config.ID = id
		}
		return updateArtifactNotificationConfigurationLinks(ctx, config)
	})
}

func UpdateArtifactNotificationConfiguration(
	ctx context.Context,
	config *types.ArtifactNotificationConfiguration,
) error {
	return RunTxRR(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)
		cmd, err := db.Exec(
			ctx,
			`UPDATE ArtifactNotificationConfiguration SET
				name = @name,
				enabled = @enabled,
				new_version_trigger_enabled = @newVersionTriggerEnabled,
				customer_message = @customerMessage
			WHERE id = @id
				AND organization_id = @organizationID
				AND ((@customerOrgIsNull AND customer_organization_id IS NULL)
					OR customer_organization_id = @customerOrganizationID)`,
			artifactNotificationConfigurationArgs(config),
		)
		if err != nil {
			return fmt.Errorf("failed to update ArtifactNotificationConfiguration: %w", err)
		} else if cmd.RowsAffected() == 0 {
			return apierrors.ErrNotFound
		}
		return updateArtifactNotificationConfigurationLinks(ctx, config)
	})
}

func artifactNotificationConfigurationArgs(config *types.ArtifactNotificationConfiguration) pgx.NamedArgs {
	return pgx.NamedArgs{
		"id":                       config.ID,
		"organizationID":           config.OrganizationID,
		"customerOrganizationID":   config.CustomerOrganizationID,
		"customerOrgIsNull":        config.CustomerOrganizationID == nil,
		"name":                     config.Name,
		"enabled":                  config.Enabled,
		"newVersionTriggerEnabled": config.NewVersionTriggerEnabled,
		"customerMessage":          config.CustomerMessage,
	}
}

func updateArtifactNotificationConfigurationLinks(
	ctx context.Context,
	config *types.ArtifactNotificationConfiguration,
) error {
	if err := replaceNotificationLinks(ctx, notificationLink{
		table:        "ArtifactNotificationConfiguration_Artifact",
		configColumn: "artifact_notification_configuration_id",
		valueColumn:  "artifact_id",
		valueTable:   "Artifact",
		// Every version of a mirrored artifact appears at once when the upstream sync runs, so
		// announcing them would mean a mail per upstream tag.
		valueCondition: "v.upstream_url IS NULL",
	}, config.ID, config.OrganizationID, config.ArtifactIDs); err != nil {
		return err
	}
	if err := replaceNotificationRecipients(
		ctx,
		"ArtifactNotificationConfiguration_Organization_UserAccount",
		"artifact_notification_configuration_id",
		config.ID,
		config.OrganizationID,
		config.UserAccountIDs,
	); err != nil {
		return err
	}
	return getArtifactNotificationConfigurationInto(ctx, config.ID, config)
}

func getArtifactNotificationConfigurationInto(
	ctx context.Context,
	id uuid.UUID,
	target *types.ArtifactNotificationConfiguration,
) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+artifactNotificationConfigurationOutputExpr+`
		FROM ArtifactNotificationConfiguration c WHERE c.id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return fmt.Errorf("failed to query ArtifactNotificationConfiguration: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(
		rows,
		pgx.RowToStructByName[types.ArtifactNotificationConfiguration],
	)
	if err != nil {
		return fmt.Errorf("failed to collect ArtifactNotificationConfiguration: %w", err)
	}
	*target = result
	return nil
}

func DeleteArtifactNotificationConfiguration(
	ctx context.Context,
	id, organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) error {
	return deleteNotificationConfiguration(
		ctx, "ArtifactNotificationConfiguration", id, organizationID, customerOrganizationID,
	)
}

func deleteNotificationConfiguration(
	ctx context.Context,
	table string,
	id, organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(
		ctx,
		fmt.Sprintf(`DELETE FROM %v
		WHERE id = @id
			AND organization_id = @organizationID
			AND ((@customerOrgIsNull AND customer_organization_id IS NULL)
				OR customer_organization_id = @customerOrganizationID)`, table),
		pgx.NamedArgs{
			"id":                     id,
			"organizationID":         organizationID,
			"customerOrganizationID": customerOrganizationID,
			"customerOrgIsNull":      customerOrganizationID == nil,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to delete %v: %w", table, err)
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}
