package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	deploymentTargetOutputExprBase = `
		dt.id,
		dt.created_at,
		dt.name,
		dt.type,
		dt.access_key_salt,
		dt.access_key_hash,
		dt.namespace,
		dt.scope,
		dt.organization_id,
		dt.customer_organization_id,
		dt.agent_version_id,
		dt.reported_agent_version_id,
		dt.metrics_enabled,
		CASE WHEN dt.resources_cpu_request IS NOT NULL THEN (
			dt.resources_cpu_request,
			dt.resources_memory_request,
			dt.resources_cpu_limit,
			dt.resources_memory_limit
		) END
	`
	deploymentTargetOutputExpr = deploymentTargetOutputExprBase +
		", CASE WHEN co.id IS NOT NULL THEN (" + customerOrganizationOutputExpr + ") END AS customer_organization"
	deploymentTargetFullOutputExpr = deploymentTargetOutputExpr + `,
		CASE WHEN status.id IS NOT NULL
			THEN (status.id, status.created_at, status.message) END
			AS current_status,
		CASE WHEN agv.id IS NOT NULL
			THEN (agv.id, agv.created_at, agv.name, agv.manifest_file_revision, agv.compose_file_revision) END
			AS agent_version
	`
	deploymentTargetJoinExpr = `
		LEFT JOIN (
			-- find the creation date of the latest status entry for each deployment target
			-- IMPORTANT: The sub-query here might seem inefficient but it is MUCH FASTER than using a GROUP BY clause
			-- because it can utilize an index!!
			SELECT
				dt1.id AS deployment_target_id,
				(SELECT max(created_at) FROM DeploymentTargetStatus WHERE deployment_target_id = dt1.id) AS max_created_at
			FROM DeploymentTarget dt1
		) status_max
		 	ON dt.id = status_max.deployment_target_id
		LEFT JOIN DeploymentTargetStatus status
			ON dt.id = status.deployment_target_id
			AND status.created_at = status_max.max_created_at
		LEFT JOIN AgentVersion agv
			ON dt.agent_version_id = agv.id
		LEFT JOIN CustomerOrganization co
			ON dt.customer_organization_id = co.id
		LEFT JOIN Organization o
			ON o.id = dt.organization_id AND o.deleted_at IS NULL
	`
	deploymentTargetFromExpr = `
		DeploymentTarget dt
	` + deploymentTargetJoinExpr
)

func GetDeploymentTargets(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrgID *uuid.UUID,
) ([]types.DeploymentTargetFull, error) {
	db := internalctx.GetDb(ctx)
	isVendor := customerOrgID == nil
	if rows, err := db.Query(ctx,
		"SELECT"+deploymentTargetFullOutputExpr+"FROM"+deploymentTargetFromExpr+
			"WHERE dt.organization_id = @orgId "+
			"AND (@isVendor OR dt.customer_organization_id = @customerOrgId) "+
			"ORDER BY co.name, dt.name",
		pgx.NamedArgs{"orgId": orgID, "customerOrgId": customerOrgID, "isVendor": isVendor},
	); err != nil {
		return nil, fmt.Errorf("failed to query DeploymentTargets: %w", err)
	} else if result, err := pgx.CollectRows(
		rows,
		pgx.RowToStructByPos[types.DeploymentTargetFull],
	); err != nil {
		return nil, fmt.Errorf("failed to get DeploymentTargets: %w", err)
	} else {
		for i := range result {
			if err := addDeploymentsToTarget(ctx, &result[i]); err != nil {
				return nil, err
			}
		}
		return result, nil
	}
}

func CountDeploymentTargets(ctx context.Context, orgID uuid.UUID, customerOrgID *uuid.UUID) (int64, error) {
	db := internalctx.GetDb(ctx)

	customerOwned := customerOrgID == nil

	rows, err := db.Query(ctx,
		`SELECT count(*)
		FROM DeploymentTarget
		WHERE organization_id = @orgId
			AND (customer_organization_id = @customerOrgId
		    OR ( @customerOwned AND customer_organization_id is null))`,
		pgx.NamedArgs{"orgId": orgID, "customerOrgId": customerOrgID, "customerOwned": customerOwned},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to count DeploymentTargets: %w", err)
	}

	if count, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int64]); err != nil {
		return 0, fmt.Errorf("failed to count DeploymentTargets: %w", err)
	} else {
		return count, nil
	}
}

func GetDeploymentTarget(
	ctx context.Context,
	id uuid.UUID,
	orgID *uuid.UUID,
) (*types.DeploymentTargetFull, error) {
	db := internalctx.GetDb(ctx)
	var args pgx.NamedArgs
	query := "SELECT" + deploymentTargetFullOutputExpr + "FROM" + deploymentTargetFromExpr +
		" WHERE dt.id = @id "
	if orgID != nil {
		args = pgx.NamedArgs{"id": id, "orgId": *orgID, "checkOrg": true}
		query += " AND dt.organization_id = @orgId"
	} else {
		args = pgx.NamedArgs{"id": id, "checkOrg": false}
	}
	rows, err := db.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentTargets: %w", err)
	}
	result, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[types.DeploymentTargetFull])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get DeploymentTarget: %w", err)
	} else {
		return &result, addDeploymentsToTarget(ctx, &result)
	}
}

func GetDeploymentTargetForDeploymentID(
	ctx context.Context,
	id uuid.UUID,
) (*types.DeploymentTargetFull, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		fmt.Sprintf("SELECT %v FROM %v JOIN Deployment d ON dt.id = d.deployment_target_id WHERE d.id = @id ",
			deploymentTargetFullOutputExpr, deploymentTargetFromExpr),
		pgx.NamedArgs{"id": id},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentTargets: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[types.DeploymentTargetFull])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get DeploymentTarget: %w", err)
	} else {
		return &result, addDeploymentsToTarget(ctx, &result)
	}
}

func CreateDeploymentTarget(
	ctx context.Context,
	dt *types.DeploymentTargetFull,
	orgID, createdByID uuid.UUID,
	customerOrgID *uuid.UUID,
) error {
	dt.OrganizationID = orgID

	db := internalctx.GetDb(ctx)
	args := pgx.NamedArgs{
		"name":           dt.Name,
		"type":           dt.Type,
		"orgId":          dt.OrganizationID,
		"namespace":      dt.Namespace,
		"scope":          dt.Scope,
		"agentVersionId": dt.AgentVersionID,
		"metricsEnabled": dt.MetricsEnabled,
		"customerOrgId":  customerOrgID,
	}

	if dt.Resources != nil {
		args["resourcesCpuRequest"] = dt.Resources.CPURequest
		args["resourcesMemoryRequest"] = dt.Resources.MemoryRequest
		args["resourcesCpuLimit"] = dt.Resources.CPULimit
		args["resourcesMemoryLimit"] = dt.Resources.MemoryLimit
	}

	rows, err := db.Query(
		ctx,
		`WITH inserted AS (
			INSERT INTO DeploymentTarget
			(name, type, organization_id, namespace, scope, agent_version_id, metrics_enabled,
				customer_organization_id, resources_cpu_request, resources_memory_request, resources_cpu_limit,
				resources_memory_limit)
			VALUES (@name, @type, @orgId, @namespace, @scope, @agentVersionId, @metricsEnabled, @customerOrgId,
				@resourcesCpuRequest, @resourcesMemoryRequest, @resourcesCpuLimit, @resourcesMemoryLimit)
			RETURNING *
		)
		SELECT `+deploymentTargetFullOutputExpr+` FROM inserted dt`+deploymentTargetJoinExpr,
		args,
	)
	if err != nil {
		return fmt.Errorf("failed to query DeploymentTargets: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[types.DeploymentTargetFull])
	if err != nil {
		return fmt.Errorf("could not save DeploymentTarget: %w", err)
	} else {
		*dt = result
		return addDeploymentsToTarget(ctx, dt)
	}
}

func UpdateDeploymentTarget(ctx context.Context, dt *types.DeploymentTargetFull, orgID uuid.UUID) error {
	agentUpdateStr := ""
	db := internalctx.GetDb(ctx)
	args := pgx.NamedArgs{
		"id":             dt.ID,
		"name":           dt.Name,
		"orgId":          orgID,
		"metricsEnabled": dt.MetricsEnabled,
	}
	if dt.AgentVersionID != nil {
		args["agentVersionId"] = dt.AgentVersionID
		agentUpdateStr = ", agent_version_id = @agentVersionId "
	}
	if dt.Resources != nil {
		args["cpuRequest"] = dt.Resources.CPURequest
		args["cpuLimit"] = dt.Resources.CPULimit
		args["memoryRequest"] = dt.Resources.MemoryRequest
		args["memoryLimit"] = dt.Resources.MemoryLimit
	}
	rows, err := db.Query(ctx,
		`WITH updated AS (
			UPDATE DeploymentTarget AS dt SET
				name = @name,
				metrics_enabled = @metricsEnabled,
				resources_cpu_request = @cpuRequest,
				resources_cpu_limit = @cpuLimit,
				resources_memory_request = @memoryRequest,
				resources_memory_limit = @memoryLimit `+agentUpdateStr+`
			WHERE id = @id AND organization_id = @orgId RETURNING *
		)
		SELECT `+deploymentTargetFullOutputExpr+` FROM updated dt`+deploymentTargetJoinExpr,
		args)
	if err != nil {
		return fmt.Errorf("could not update DeploymentTarget: %w", err)
	} else if updated, err := pgx.CollectExactlyOneRow(
		rows, pgx.RowToStructByPos[types.DeploymentTargetFull],
	); err != nil {
		return fmt.Errorf("could not get updated DeploymentTarget: %w", err)
	} else {
		*dt = updated
		return addDeploymentsToTarget(ctx, dt)
	}
}

func DeleteDeploymentTargetWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	if cmd, err := db.Exec(ctx, `DELETE FROM DeploymentTarget WHERE id = @id`, pgx.NamedArgs{"id": id}); err != nil {
		return err
	} else if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	} else {
		return nil
	}
}

func UpdateDeploymentTargetAccess(ctx context.Context, dt *types.DeploymentTarget, orgID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"UPDATE DeploymentTarget AS dt SET access_key_salt = @accessKeySalt, access_key_hash = @accessKeyHash "+
			"WHERE id = @id AND organization_id = @orgId RETURNING "+
			deploymentTargetOutputExprBase,
		pgx.NamedArgs{"accessKeySalt": dt.AccessKeySalt, "accessKeyHash": dt.AccessKeyHash, "id": dt.ID, "orgId": orgID})
	if err != nil {
		return fmt.Errorf("could not update DeploymentTarget: %w", err)
	} else if updated, err := pgx.CollectExactlyOneRow(
		rows, pgx.RowToStructByPos[types.DeploymentTarget],
	); err != nil {
		return fmt.Errorf("could not get updated DeploymentTarget: %w", err)
	} else {
		*dt = updated
		return nil
	}
}

func UpdateDeploymentTargetReportedAgentVersionID(
	ctx context.Context,
	dt *types.DeploymentTargetFull,
	agentVersionID uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH updated AS (
			UPDATE DeploymentTarget AS dt
			SET reported_agent_version_id = @agentVersionId
			WHERE id = @id
			RETURNING *
		)
		SELECT`+deploymentTargetFullOutputExpr+`FROM updated dt`+deploymentTargetJoinExpr,
		pgx.NamedArgs{"id": dt.ID, "agentVersionId": agentVersionID},
	)
	if err != nil {
		return err
	} else if updated, err := pgx.CollectExactlyOneRow(rows,
		pgx.RowToAddrOfStructByName[types.DeploymentTargetFull]); err != nil {
		return err
	} else {
		*dt = *updated
		return nil
	}
}

func CreateDeploymentTargetStatus(ctx context.Context, dt *types.DeploymentTarget, message string) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		"INSERT INTO DeploymentTargetStatus (deployment_target_id, message) VALUES (@deploymentTargetId, @message)",
		pgx.NamedArgs{"deploymentTargetId": dt.ID, "message": message})
	if err != nil {
		return err
	} else {
		rows.Close()
		return rows.Err()
	}
}

func CleanupDeploymentTargetStatus(ctx context.Context) (int64, error) {
	if env.StatusEntriesMaxAge() == nil {
		return 0, nil
	}
	db := internalctx.GetDb(ctx)
	if cmd, err := db.Exec(
		ctx,
		`DELETE FROM DeploymentTargetStatus dts
		USING (
			SELECT
				dt.id AS deployment_target_id,
				(SELECT max(created_at) FROM DeploymentTargetStatus WHERE deployment_target_id = dt.id)
					AS max_created_at
			FROM DeploymentTarget dt
		) max_created_at
		WHERE dts.deployment_target_id = max_created_at.deployment_target_id
			AND dts.created_at < max_created_at.max_created_at
			AND current_timestamp - dts.created_at > @statusEntriesMaxAge`,
		pgx.NamedArgs{"statusEntriesMaxAge": env.StatusEntriesMaxAge()},
	); err != nil {
		return 0, err
	} else {
		return cmd.RowsAffected(), nil
	}
}

func addDeploymentsToTarget(ctx context.Context, dt *types.DeploymentTargetFull) error {
	if d, err := GetDeploymentsForDeploymentTarget(ctx, dt.ID); errors.Is(err, apierrors.ErrNotFound) {
		return nil
	} else if err != nil {
		return err
	} else {
		dt.Deployments = d
		return nil
	}
}
