package db

import (
	"context"
	"fmt"

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
	r.deployment_revision_id,
	r.deployment_status_message,
	r.resolved_at,
	r.metric_type,
	r.disk_device,
	r.disk_path,
	r.previous_deployment_target_metrics_id,
	r.current_deployment_target_metrics_id,
	r.delivery_error `

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
				deployment_revision_id,
				deployment_status_message,
				metric_type,
				disk_device,
				disk_path,
				previous_deployment_target_metrics_id,
				current_deployment_target_metrics_id,
				delivery_error
			)
			VALUES (
				@organizationID,
				@customerOrganizationID,
				@deploymentTargetID,
				@alertConfigurationID,
				@type,
				@deploymentRevisionID,
				@deploymentStatusMessage,
				@metricType,
				@diskDevice,
				@diskPath,
				@previousMetricsID,
				@currentMetricsID,
				@deliveryError
			)
			RETURNING *
		)
		SELECT`+notificationRecordOutputExpr+`FROM inserted r`,
		pgx.NamedArgs{
			"organizationID":          record.OrganizationID,
			"customerOrganizationID":  record.CustomerOrganizationID,
			"deploymentTargetID":      record.DeploymentTargetID,
			"alertConfigurationID":    record.AlertConfigurationID,
			"type":                    record.Type,
			"deploymentRevisionID":    record.DeploymentRevisionID,
			"deploymentStatusMessage": record.DeploymentStatusMessage,
			"metricType":              record.MetricType,
			"diskDevice":              record.DiskDevice,
			"diskPath":                record.DiskPath,
			"previousMetricsID":       record.PreviousDeploymentTargetMetricsID,
			"currentMetricsID":        record.CurrentDeploymentTargetMetricsID,
			"deliveryError":           record.DeliveryError,
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

func HasOpenStaleWarning(ctx context.Context, configID, deploymentID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM NotificationRecord r
			JOIN DeploymentRevision dr ON dr.id = r.deployment_revision_id
			WHERE r.alert_configuration_id = @alertConfigurationID
				AND dr.deployment_id = @deploymentID
				AND r.type = 'warning'
				AND r.resolved_at IS NULL
		)`,
		pgx.NamedArgs{"alertConfigurationID": configID, "deploymentID": deploymentID},
	)
	if err != nil {
		return false, fmt.Errorf("failed to query open stale warning: %w", err)
	}

	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, fmt.Errorf("failed to collect open stale warning: %w", err)
	}
	return exists, nil
}

// ResolveStaleWarnings resolves the open stale warnings of every alert configuration for the deployment and returns
// the configurations they belonged to. Only one caller can resolve a warning, so it is the one that has to send the
// recovery notification.
func ResolveStaleWarnings(ctx context.Context, deploymentID uuid.UUID) ([]uuid.UUID, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH resolved AS (
			UPDATE NotificationRecord r SET resolved_at = now()
			FROM DeploymentRevision dr
			WHERE dr.id = r.deployment_revision_id
				AND dr.deployment_id = @deploymentID
				AND r.type = 'warning'
				AND r.resolved_at IS NULL
			RETURNING r.alert_configuration_id
		)
		SELECT DISTINCT alert_configuration_id FROM resolved WHERE alert_configuration_id IS NOT NULL`,
		pgx.NamedArgs{"deploymentID": deploymentID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve stale warnings: %w", err)
	}

	configIDs, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("failed to collect resolved stale warnings: %w", err)
	}
	return configIDs, nil
}

func GetNotificationRecords(
	ctx context.Context,
	organizationID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]types.NotificationRecordWithDetails, error) {
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
			ON dr.id = r.deployment_revision_id
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

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.NotificationRecordWithDetails])
	if err != nil {
		return nil, fmt.Errorf("failed to collect NotificationRecord: %w", err)
	}

	return records, nil
}
