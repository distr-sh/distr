package db

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func GetDeploymentTargetsForMetrics(ctx context.Context) ([]types.DeploymentTargetStatusMetricsItem, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT o.name, co.name, dt.name, status.created_at FROM`+deploymentTargetFromExpr+`WHERE o.deleted_at IS NULL`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentTargets: %w", err)
	}

	result, err := pgx.CollectRows(rows, pgx.RowToStructByPos[types.DeploymentTargetStatusMetricsItem])
	if err != nil {
		return nil, fmt.Errorf("failed to collect DeploymentTargets: %w", err)
	}

	return result, nil
}

func GetDeploymentTargetForMetricsByID(
	ctx context.Context,
	id uuid.UUID,
) (*types.DeploymentTargetStatusMetricsItem, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT o.name, co.name, dt.name, status.created_at FROM`+deploymentTargetFromExpr+`WHERE dt.id = @id`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentTargets: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByPos[types.DeploymentTargetStatusMetricsItem])
	if err != nil {
		return nil, fmt.Errorf("failed to collect DeploymentTargets: %w", err)
	}

	return result, nil
}

func GetDeploymentsForMetrics(ctx context.Context) ([]types.DeploymentStatusMetricsItem, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT o.name, co.name, dt.name, d.id, a.name, av.name, drs.created_at, drs.type
		FROM`+deploymentWithLatestRevisionFromExpr+`
		JOIN DeploymentTarget dt ON d.deployment_target_id = dt.id
		LEFT JOIN CustomerOrganization co ON dt.customer_organization_id = co.id
		JOIN Organization o ON dt.organization_id = o.id
		WHERE o.deleted_at IS NULL`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Deployments: %w", err)
	}

	result, err := pgx.CollectRows(rows, pgx.RowToStructByPos[types.DeploymentStatusMetricsItem])
	if err != nil {
		return nil, fmt.Errorf("failed to collect Deployments: %w", err)
	}

	return result, nil
}

func GetDeploymentForMetricsByRevisionID(
	ctx context.Context,
	id uuid.UUID,
) (*types.DeploymentStatusMetricsItem, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT o.name, co.name, dt.name, d.id, a.name, av.name, drs.created_at, drs.type
		FROM`+deploymentWithLatestRevisionFromExpr+`
		JOIN DeploymentTarget dt ON d.deployment_target_id = dt.id
		LEFT JOIN CustomerOrganization co ON dt.customer_organization_id = co.id
		JOIN DeploymentRevision dr1 ON dr1.deployment_id = d.id AND dr1.id = @id
		JOIN Organization o ON dt.organization_id = o.id
		WHERE o.deleted_at IS NULL`,
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Deployments: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByPos[types.DeploymentStatusMetricsItem])
	if err != nil {
		return nil, fmt.Errorf("failed to collect Deployments: %w", err)
	}

	return result, nil
}

// QueryableInitDataSource provides a prometheus.InitDataSource implementation based on regular DB functions
type QueryableInitDataSource struct{}

func (QueryableInitDataSource) OrganizationsTotal(ctx context.Context) (int64, error) {
	return CountAllOrganizations(ctx)
}

func (QueryableInitDataSource) DeploymentTargetStatus(
	ctx context.Context,
) ([]types.DeploymentTargetStatusMetricsItem, error) {
	return GetDeploymentTargetsForMetrics(ctx)
}

func (QueryableInitDataSource) DeploymentStatus(ctx context.Context) ([]types.DeploymentStatusMetricsItem, error) {
	return GetDeploymentsForMetrics(ctx)
}
