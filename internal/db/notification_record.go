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

const notificationRecordOutputExpr = `
	r.id,
	r.created_at,
	r.organization_id,
	r.customer_organization_id,
	r.deployment_target_id,
	r.deployment_status_notification_configuration_id,
	r.previous_deployment_revision_status_id,
	r.current_deployment_revision_status_id,
	r.message `

func SaveNotificationRecord(ctx context.Context, record *types.NotificationRecord) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH inserted AS (
			INSERT INTO NotificationRecord (
				organization_id,
				customer_organization_id,
				deployment_target_id,
				deployment_status_notification_configuration_id,
				previous_deployment_revision_status_id,
				current_deployment_revision_status_id,
				message
			)
			VALUES (
				@organizationID,
				@customerOrganizationID,
				@deploymentTargetID,
				@deploymentStatusNotificationConfigurationID,
				@previousDeploymentStatusID,
				@currentDeploymentStatusID,
				@message
			)
			RETURNING *
		)
		SELECT`+notificationRecordOutputExpr+`FROM inserted r`,
		pgx.NamedArgs{
			"organizationID":                              record.OrganizationID,
			"customerOrganizationID":                      record.CustomerOrganizationID,
			"deploymentTargetID":                          record.DeploymentTargetID,
			"deploymentStatusNotificationConfigurationID": record.DeploymentStatusNotificationConfigurationID,
			"previousDeploymentStatusID":                  record.PreviousDeploymentRevisionStatusID,
			"currentDeploymentStatusID":                   record.CurrentDeploymentRevisionStatusID,
			"message":                                     record.Message,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to save NotificationRecord: %w", err)
	}

	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.NotificationRecord]); err != nil {
		return fmt.Errorf("failed to collect NotificationRecord: %w", err)
	} else {
		*record = result
	}

	return nil
}

func GetLatestNotificationRecord(
	ctx context.Context,
	configID, previousID uuid.UUID,
) (*types.NotificationRecord, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+notificationRecordOutputExpr+`FROM NotificationRecord r
		WHERE r.deployment_status_notification_configuration_id = @deploymentStatusNotificationConfigurationID
			AND r.previous_deployment_revision_status_id = @previousDeploymentStatusID
		ORDER BY r.created_at DESC LIMIT 1`,
		pgx.NamedArgs{
			"deploymentStatusNotificationConfigurationID": configID,
			"previousDeploymentStatusID":                  previousID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query NotificationRecord exists: %w", err)
	}

	if record, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.NotificationRecord]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to collect NotificationRecord exists: %w", err)
	} else {
		return &record, nil
	}
}

func GetNotificationRecords(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]types.NotificationRecordWithCurrentStatus, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT`+notificationRecordOutputExpr+`,
			dt.name AS deployment_target_name,
			CASE WHEN s.id IS NOT NULL THEN (
				s.id, s.created_at, s.deployment_revision_id, s.type, s.message
			) END current_deployment_revision_status
		FROM NotificationRecord r
		LEFT JOIN DeploymentTarget dt ON r.deployment_target_id = dt.id
		LEFT JOIN DeploymentRevisionStatus s
			ON r.current_deployment_revision_status_id = s.id
		WHERE r.organization_id = @organizationID
			AND ((@isVendor AND r.customer_organization_id IS NULL)
				OR r.customer_organization_id = @customerOrganizationID)
		ORDER BY r.created_at DESC`,
		pgx.NamedArgs{
			"organizationID":         organizationID,
			"customerOrganizationID": customerOrganizationID,
			"isVendor":               customerOrganizationID == nil,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query NotificationRecord exists: %w", err)
	}

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.NotificationRecordWithCurrentStatus])
	if err != nil {
		return nil, fmt.Errorf("failed to collect NotificationRecord: %w", err)
	}

	return records, nil
}
