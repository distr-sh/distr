package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const environmentOutputExpr = `
	e.id,
	e.created_at,
	e.updated_at,
	e.organization_id,
	e.name,
	e.description,
	e.sort_order,
	e.is_production,
	e.allow_dynamic_targets,
	e.retention_policy_id
`

func CreateEnvironment(ctx context.Context, environment *types.Environment) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`INSERT INTO Environment AS e (
			organization_id,
			name,
			description,
			sort_order,
			is_production,
			allow_dynamic_targets,
			retention_policy_id
		) VALUES (
			@organizationId,
			@name,
			@description,
			@sortOrder,
			@isProduction,
			@allowDynamicTargets,
			@retentionPolicyId
		) RETURNING `+environmentOutputExpr,
		pgx.NamedArgs{
			"organizationId":      environment.OrganizationID,
			"name":                environment.Name,
			"description":         environment.Description,
			"sortOrder":           environment.SortOrder,
			"isProduction":        environment.IsProduction,
			"allowDynamicTargets": environment.AllowDynamicTargets,
			"retentionPolicyId":   environment.RetentionPolicyID,
		},
	)
	if err != nil {
		return mapEnvironmentWriteError("insert", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Environment])
	if err != nil {
		return mapEnvironmentWriteError("scan created", err)
	}
	*environment = result
	return nil
}

func GetEnvironmentsByOrganizationID(ctx context.Context, orgID uuid.UUID) ([]types.Environment, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT `+environmentOutputExpr+`
		FROM Environment e
		WHERE e.organization_id = @organizationId
		ORDER BY e.sort_order, e.name`,
		pgx.NamedArgs{"organizationId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query Environment: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Environment])
	if err != nil {
		return nil, fmt.Errorf("could not collect Environment: %w", err)
	}
	return result, nil
}

func GetEnvironment(ctx context.Context, id, orgID uuid.UUID) (*types.Environment, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT `+environmentOutputExpr+`
		FROM Environment e
		WHERE e.id = @id AND e.organization_id = @organizationId`,
		pgx.NamedArgs{"id": id, "organizationId": orgID},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query Environment: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Environment])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("could not collect Environment: %w", err)
	}
	return &result, nil
}

func UpdateEnvironment(ctx context.Context, environment *types.Environment) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`UPDATE Environment AS e SET
			name = @name,
			description = @description,
			sort_order = @sortOrder,
			is_production = @isProduction,
			allow_dynamic_targets = @allowDynamicTargets,
			retention_policy_id = @retentionPolicyId,
			updated_at = now()
		WHERE e.id = @id AND e.organization_id = @organizationId
		RETURNING `+environmentOutputExpr,
		pgx.NamedArgs{
			"id":                  environment.ID,
			"organizationId":      environment.OrganizationID,
			"name":                environment.Name,
			"description":         environment.Description,
			"sortOrder":           environment.SortOrder,
			"isProduction":        environment.IsProduction,
			"allowDynamicTargets": environment.AllowDynamicTargets,
			"retentionPolicyId":   environment.RetentionPolicyID,
		},
	)
	if err != nil {
		return mapEnvironmentWriteError("update", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Environment])
	if errors.Is(err, pgx.ErrNoRows) {
		return apierrors.ErrNotFound
	} else if err != nil {
		return mapEnvironmentWriteError("scan updated", err)
	}
	*environment = result
	return nil
}

func DeleteEnvironmentWithID(ctx context.Context, id, organizationID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`DELETE FROM Environment WHERE id = @id AND organization_id = @organizationId`,
		pgx.NamedArgs{"id": id, "organizationId": organizationID},
	)
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.ForeignKeyViolation {
			return fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return fmt.Errorf("could not delete Environment: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apierrors.ErrNotFound
	}
	return nil
}

func mapEnvironmentWriteError(action string, err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
		return fmt.Errorf("could not %s Environment: %w", action, apierrors.ErrAlreadyExists)
	}
	return fmt.Errorf("could not %s Environment: %w", action, err)
}
