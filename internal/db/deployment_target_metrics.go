package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/distr-sh/distr/api"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DeploymentTargetLatestMetrics struct {
	ID uuid.UUID `db:"id" json:"id"`
	api.AgentDeploymentTargetMetrics
}

func GetLatestDeploymentTargetMetrics(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) ([]DeploymentTargetLatestMetrics, error) {
	db := internalctx.GetDb(ctx)
	isVendorUser := customerOrganizationID == nil

	rows, err := db.Query(ctx,
		`SELECT dt.id, dtm.cpu_cores_millis, dtm.cpu_usage, dtm.memory_bytes, dtm.memory_usage,
			json_agg(json_build_object(
				'device', dtdm.device,
				'path', dtdm.path,
				'fsType', dtdm.fs_type,
				'bytesTotal', dtdm.bytes_total,
				'bytesUsed', dtdm.bytes_used
			)) FILTER (WHERE dtdm.id IS NOT NULL) AS disk_metrics
		FROM DeploymentTarget dt
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
		GROUP BY dt.id, dtm.id, co.name, dt.name
		ORDER BY co.name, dt.name`,
		pgx.NamedArgs{"orgId": orgID, "customerOrganizationId": customerOrganizationID, "isVendorUser": isVendorUser},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentTargets: %w", err)
	}
	defer rows.Close()

	var result []DeploymentTargetLatestMetrics
	for rows.Next() {
		var m DeploymentTargetLatestMetrics
		var diskJSON []byte
		if err := rows.Scan(&m.ID, &m.CPUCoresMillis, &m.CPUUsage, &m.MemoryBytes, &m.MemoryUsage, &diskJSON); err != nil {
			return nil, fmt.Errorf("failed to scan DeploymentTargetMetrics row: %w", err)
		}
		if diskJSON != nil {
			if err := json.Unmarshal(diskJSON, &m.DiskMetrics); err != nil {
				return nil, fmt.Errorf("failed to unmarshal disk metrics: %w", err)
			}
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to get DeploymentTargets: %w", err)
	}
	return result, nil
}

func CreateDeploymentTargetMetrics(
	ctx context.Context,
	dt *types.DeploymentTarget,
	metrics *api.AgentDeploymentTargetMetrics,
) error {
	db := internalctx.GetDb(ctx)

	var metricsID uuid.UUID
	err := db.QueryRow(ctx,
		"INSERT INTO DeploymentTargetMetrics "+
			"(deployment_target_id, cpu_cores_millis, cpu_usage, memory_bytes, memory_usage) "+
			"VALUES (@deploymentTargetId, @cpuCoresMillis, @cpuUsage, @memoryBytes, @memoryUsage) "+
			"RETURNING id",
		pgx.NamedArgs{
			"deploymentTargetId": dt.ID,
			"cpuCoresMillis":     metrics.CPUCoresMillis,
			"cpuUsage":           metrics.CPUUsage,
			"memoryBytes":        metrics.MemoryBytes,
			"memoryUsage":        metrics.MemoryUsage,
		}).Scan(&metricsID)
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
			return []any{metricsID, d.Device, d.Path, d.FsType, d.BytesTotal, d.BytesUsed}, nil
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
