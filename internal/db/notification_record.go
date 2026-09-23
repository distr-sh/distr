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
	"github.com/jackc/pgx/v5"
)

const notificationRecordOutputExpr = `
	r.id,
	r.created_at,
	r.organization_id,
	r.customer_organization_id,
	r.deployment_target_id,
	r.alert_configuration_id,
	r.type,
	r.previous_deployment_revision_id,
	r.previous_status_created_at,
	r.current_deployment_revision_id,
	r.current_status_created_at,
	r.current_status_type,
	r.current_status_message,
	r.metric_type,
	r.disk_device,
	r.disk_path,
	r.previous_deployment_target_metrics_id,
	r.current_deployment_target_metrics_id,
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
				alert_configuration_id,
				type,
				previous_deployment_revision_id,
				previous_status_created_at,
				current_deployment_revision_id,
				current_status_created_at,
				current_status_type,
				current_status_message,
				metric_type,
				disk_device,
				disk_path,
				previous_deployment_target_metrics_id,
				current_deployment_target_metrics_id,
				message
			)
			VALUES (
				@organizationID,
				@customerOrganizationID,
				@deploymentTargetID,
				@alertConfigurationID,
				@type,
				@previousDeploymentRevisionID,
				@previousStatusCreatedAt,
				@currentDeploymentRevisionID,
				@currentStatusCreatedAt,
				@currentStatusType,
				@currentStatusMessage,
				@metricType,
				@diskDevice,
				@diskPath,
				@previousMetricsID,
				@currentMetricsID,
				@message
			)
			RETURNING *
		)
		SELECT`+notificationRecordOutputExpr+`FROM inserted r`,
		pgx.NamedArgs{
			"organizationID":               record.OrganizationID,
			"customerOrganizationID":       record.CustomerOrganizationID,
			"deploymentTargetID":           record.DeploymentTargetID,
			"alertConfigurationID":         record.AlertConfigurationID,
			"type":                         record.Type,
			"previousDeploymentRevisionID": record.PreviousDeploymentRevisionID,
			"previousStatusCreatedAt":      record.PreviousStatusCreatedAt,
			"currentDeploymentRevisionID":  record.CurrentDeploymentRevisionID,
			"currentStatusCreatedAt":       record.CurrentStatusCreatedAt,
			"currentStatusType":            record.CurrentStatusType,
			"currentStatusMessage":         record.CurrentStatusMessage,
			"metricType":                   record.MetricType,
			"diskDevice":                   record.DiskDevice,
			"diskPath":                     record.DiskPath,
			"previousMetricsID":            record.PreviousDeploymentTargetMetricsID,
			"currentMetricsID":             record.CurrentDeploymentTargetMetricsID,
			"message":                      record.Message,
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
	configID, previousRevisionID uuid.UUID,
	previousStatusCreatedAt time.Time,
) (*types.NotificationRecord, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+notificationRecordOutputExpr+`FROM NotificationRecord r
		WHERE r.alert_configuration_id = @alertConfigurationID
			AND r.previous_deployment_revision_id = @previousDeploymentRevisionID
			AND r.previous_status_created_at = @previousStatusCreatedAt
		ORDER BY r.created_at DESC LIMIT 1`,
		pgx.NamedArgs{
			"alertConfigurationID":         configID,
			"previousDeploymentRevisionID": previousRevisionID,
			"previousStatusCreatedAt":      previousStatusCreatedAt,
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
			co.name AS customer_organization_name,
			a.name AS application_name,
			av.name AS application_version_name,
			CASE WHEN dtm.id IS NOT NULL THEN (
				dtm.id,
				dtm.created_at,
				dtm.deployment_target_id,
				dtm.cpu_cores_millis,
				dtm.cpu_usage,
				dtm.memory_bytes,
				dtm.memory_usage,
				array_agg(row(dtdm.device, dtdm.path, dtdm.fs_type, dtdm.bytes_total, dtdm.bytes_used) ORDER BY dtdm.device)
					FILTER (WHERE dtdm.id IS NOT NULL)
			) END AS current_deployment_target_metrics
		FROM NotificationRecord r
		LEFT JOIN DeploymentTarget dt
			ON r.deployment_target_id = dt.id
		LEFT JOIN CustomerOrganization co
			ON dt.customer_organization_id = co.id
		LEFT JOIN DeploymentRevision dr
			ON dr.id = coalesce(r.current_deployment_revision_id, r.previous_deployment_revision_id)
		LEFT JOIN ApplicationVersion av
			ON dr.application_version_id = av.id
		LEFT JOIN Application a
			ON av.application_id = a.id
		LEFT JOIN DeploymentTargetMetrics dtm
			ON r.current_deployment_target_metrics_id = dtm.id
		LEFT JOIN DeploymentTargetDiskMetrics dtdm
			ON dtdm.deployment_target_metrics_id = dtm.id
		WHERE r.organization_id = @organizationID
			AND ((@isVendor AND r.customer_organization_id IS NULL)
				OR r.customer_organization_id = @customerOrganizationID)
		GROUP BY r.id, dt.id, co.id, a.id, av.id, dtm.id
		ORDER BY r.created_at DESC`,
		pgx.NamedArgs{
			"organizationID":         organizationID,
			"customerOrganizationID": customerOrganizationID,
			"isVendor":               customerOrganizationID == nil,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query NotificationRecord: %w", err)
	}

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.NotificationRecordWithCurrentStatus])
	if err != nil {
		return nil, fmt.Errorf("failed to collect NotificationRecord: %w", err)
	}

	return records, nil
}
