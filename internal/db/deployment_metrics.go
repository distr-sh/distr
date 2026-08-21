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

const deploymentMetricsOutputExpr = `
	dm.id,
	dm.created_at,
	dm.deployment_id,
	array_agg(
		row(drm.resource, drm.container, drm.cpu_usage_millis, drm.memory_bytes, drm.cpu_limit_millis, drm.memory_limit_bytes)
		ORDER BY drm.resource, drm.container
	) FILTER (WHERE drm.id IS NOT NULL)
		AS resources
`

func CreateDeploymentMetrics(ctx context.Context, metrics *types.DeploymentMetrics) error {
	if len(metrics.Resources) == 0 {
		return nil
	}

	return RunTx(ctx, func(ctx context.Context) error {
		db := internalctx.GetDb(ctx)

		err := db.QueryRow(ctx,
			"INSERT INTO DeploymentMetrics (deployment_id) VALUES (@deploymentId) RETURNING id, created_at",
			pgx.NamedArgs{"deploymentId": metrics.DeploymentID},
		).Scan(&metrics.ID, &metrics.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert DeploymentMetrics: %w", err)
		}

		_, err = db.CopyFrom(
			ctx,
			pgx.Identifier{"deploymentresourcemetrics"},
			[]string{
				"deployment_metrics_id", "resource", "container",
				"cpu_usage_millis", "memory_bytes", "cpu_limit_millis", "memory_limit_bytes",
			},
			pgx.CopyFromSlice(len(metrics.Resources), func(i int) ([]any, error) {
				r := metrics.Resources[i]
				return []any{
					metrics.ID, r.Resource, r.Container,
					r.CPUUsageMillis, r.MemoryBytes, r.CPULimitMillis, r.MemoryLimitBytes,
				}, nil
			}),
		)
		return err
	})
}

func GetLatestDeploymentMetricsForDeploymentID(
	ctx context.Context,
	deploymentID uuid.UUID,
) (*types.DeploymentMetrics, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(ctx,
		`SELECT `+deploymentMetricsOutputExpr+` FROM (
			SELECT id, created_at, deployment_id
			FROM DeploymentMetrics
			WHERE deployment_id = @deploymentId
			ORDER BY created_at DESC, id
			LIMIT 1
		) dm
		LEFT JOIN DeploymentResourceMetrics drm ON dm.id = drm.deployment_metrics_id
		GROUP BY dm.id, dm.created_at, dm.deployment_id`,
		pgx.NamedArgs{"deploymentId": deploymentID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentMetrics: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[types.DeploymentMetrics])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to collect DeploymentMetrics: %w", err)
	}

	return result, nil
}

func CleanupDeploymentMetrics(ctx context.Context) (int64, error) {
	if env.MetricsEntriesMaxAge() == nil {
		return 0, nil
	}
	db := internalctx.GetDb(ctx)
	if cmd, err := db.Exec(
		ctx,
		`DELETE FROM DeploymentMetrics dm
		USING (
			SELECT
				d.id AS deployment_id,
				(SELECT max(created_at) FROM DeploymentMetrics WHERE deployment_id = d.id)
					AS max_created_at
			FROM Deployment d
		) max_created_at
		WHERE dm.deployment_id = max_created_at.deployment_id
			AND dm.created_at < max_created_at.max_created_at
			AND current_timestamp - dm.created_at > @metricsEntriesMaxAge`,
		pgx.NamedArgs{"metricsEntriesMaxAge": env.MetricsEntriesMaxAge()},
	); err != nil {
		return 0, err
	} else {
		return cmd.RowsAffected(), nil
	}
}
