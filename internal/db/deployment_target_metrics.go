package db

import (
	"context"
	"errors"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	deploymentTargetMetricsOutputExpr = `
		dtm.id,
		dtm.deployment_target_id AS deployment_target_id,
		dtm.cpu_cores_millis,
		dtm.cpu_usage,
		dtm.memory_bytes,
		dtm.memory_usage,
		array_agg(row(dtdm.device, dtdm.path, dtdm.fs_type, dtdm.bytes_total, dtdm.bytes_used) ORDER BY dtdm.device)
			FILTER (WHERE dtdm.id IS NOT NULL)
			AS disk_metrics
	`
)

func GetLatestDeploymentTargetMetrics(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]types.DeploymentTargetMetrics, error) {
	db := internalctx.GetDb(ctx)
	isVendorUser := customerOrganizationID == nil

	rows, err := db.Query(ctx,
		`SELECT `+deploymentTargetMetricsOutputExpr+` FROM DeploymentTarget dt
		LEFT JOIN CustomerOrganization co
			ON dt.customer_organization_id = co.id
		LEFT JOIN (
			-- copied from getting deployment target latest status:
			-- find the creation date of the latest status entry for each deployment target
			-- IMPORTANT: The sub-query here might seem inefficient but it is MUCH FASTER than using a GROUP BY clause
			-- because it can utilize an index!!
			SELECT
				dt1.id AS deployment_target_id,
				(SELECT max(created_at) FROM DeploymentTargetMetrics WHERE deployment_target_id = dt1.id) AS max_created_at
			FROM DeploymentTarget dt1
		) metrics_max
			ON dt.id = metrics_max.deployment_target_id
		INNER JOIN DeploymentTargetMetrics dtm
			ON dt.id = dtm.deployment_target_id
				AND dtm.created_at = metrics_max.max_created_at
		LEFT JOIN DeploymentTargetDiskMetrics dtdm
			ON dtm.id = dtdm.deployment_target_metrics_id
		WHERE dt.organization_id = @orgId
		AND (@isVendorUser OR dt.customer_organization_id = @customerOrganizationId)
		AND dt.metrics_enabled = true
		GROUP BY dtm.deployment_target_id, dtm.id, co.name, dt.name
		ORDER BY co.name, dt.name`,
		pgx.NamedArgs{"orgId": orgID, "customerOrganizationId": customerOrganizationID, "isVendorUser": isVendorUser},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentTargets: %w", err)
	}

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentTargetMetrics])
	if err != nil {
		return nil, fmt.Errorf("failed to collect DeploymentTargetMetrics: %w", err)
	}

	return result, nil
}

func GetLatestDeploymentTargetMetricsForID(ctx context.Context, id uuid.UUID) (*types.DeploymentTargetMetrics, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(ctx,
		`SELECT `+deploymentTargetMetricsOutputExpr+` FROM DeploymentTarget dt
		INNER JOIN DeploymentTargetMetrics dtm
			ON dt.id = dtm.deployment_target_id
		LEFT JOIN DeploymentTargetDiskMetrics dtdm
			ON dtm.id = dtdm.deployment_target_metrics_id
		WHERE dt.id = @deploymentTargetId
			AND dt.metrics_enabled = true
		GROUP BY dtm.deployment_target_id, dtm.id
		ORDER BY dtm.created_at DESC, dtm.id DESC
		LIMIT 1`,
		pgx.NamedArgs{"deploymentTargetId": id},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentTargetMetrics: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.DeploymentTargetMetrics])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to collect DeploymentTargetMetrics: %w", err)
	}

	return &result, nil
}

func CreateDeploymentTargetMetrics(
	ctx context.Context,
	metrics *types.DeploymentTargetMetrics,
) error {
	db := internalctx.GetDb(ctx)

	err := db.QueryRow(ctx,
		"INSERT INTO DeploymentTargetMetrics "+
			"(deployment_target_id, cpu_cores_millis, cpu_usage, memory_bytes, memory_usage) "+
			"VALUES (@deploymentTargetId, @cpuCoresMillis, @cpuUsage, @memoryBytes, @memoryUsage) "+
			"RETURNING id",
		pgx.NamedArgs{
			"deploymentTargetId": metrics.DeploymentTargetID,
			"cpuCoresMillis":     metrics.CPUCoresMillis,
			"cpuUsage":           metrics.CPUUsage,
			"memoryBytes":        metrics.MemoryBytes,
			"memoryUsage":        metrics.MemoryUsage,
		}).Scan(&metrics.ID)
	if err != nil {
		return err
	}

	if len(metrics.DiskMetrics) == 0 {
		return nil
	}

	_, err = db.CopyFrom(
		ctx,
		pgx.Identifier{"deploymenttargetdiskmetrics"},
		[]string{"deployment_target_metrics_id", "device", "path", "fs_type", "bytes_total", "bytes_used"},
		pgx.CopyFromSlice(len(metrics.DiskMetrics), func(i int) ([]any, error) {
			d := metrics.DiskMetrics[i]
			return []any{metrics.ID, d.Device, d.Path, d.FsType, d.BytesTotal, d.BytesUsed}, nil
		}),
	)
	return err
}

func CleanupDeploymentTargetMetrics(ctx context.Context) (int64, error) {
	if env.MetricsEntriesMaxAge() == nil {
		return 0, nil
	}
	db := internalctx.GetDb(ctx)
	if cmd, err := db.Exec(
		ctx,
		`DELETE FROM DeploymentTargetMetrics dtm
		USING (
			SELECT
				dt.id AS deployment_target_id,
				(SELECT max(created_at) FROM DeploymentTargetMetrics WHERE deployment_target_id = dt.id)
					AS max_created_at
			FROM DeploymentTarget dt
		) max_created_at
		WHERE dtm.deployment_target_id = max_created_at.deployment_target_id
			AND dtm.created_at < max_created_at.max_created_at
			AND current_timestamp - dtm.created_at > @metricsEntriesMaxAge`,
		pgx.NamedArgs{"metricsEntriesMaxAge": env.MetricsEntriesMaxAge()},
	); err != nil {
		return 0, err
	} else {
		return cmd.RowsAffected(), nil
	}
}
