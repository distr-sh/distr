package db

import (
	"context"
	"errors"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func CreateOrUpdateDeploymentTargetNotes(
	ctx context.Context,
	deploymentTargetID uuid.UUID,
	customerOrganizationID *uuid.UUID,
	userAccountID uuid.UUID,
	notes string,
) (*types.DeploymentTargetNotes, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`INSERT INTO DeploymentTargetNotes (
			deployment_target_id, customer_organization_id, updated_by_useraccount_id, updated_at, notes
		)
		VALUES (
			@deploymentTargetID, @customerOrganizationID, @userAccountID, now(), @notes
		)
		ON CONFLICT (deployment_target_id, customer_organization_id) DO UPDATE SET
			updated_by_useraccount_id = @userAccountID,
			updated_at = now(),
			notes = @notes
		RETURNING *`,
		pgx.NamedArgs{
			"deploymentTargetID":     deploymentTargetID,
			"customerOrganizationID": customerOrganizationID,
			"userAccountID":          userAccountID,
			"notes":                  notes,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DeploymentTargetNotes: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.DeploymentTargetNotes])
	if err != nil {
		return nil, fmt.Errorf("failed to collect DeploymentTargetNotes: %w", err)
	}

	return &result, nil
}

func GetDeploymentTargetNotes(
	ctx context.Context,
	deploymentTargetID uuid.UUID,
	customerOrganizationID *uuid.UUID,
) (*types.DeploymentTargetNotes, error) {
	db := internalctx.GetDb(ctx)

	customerCondition := "customer_organization_id = @customerOrganizationID"
	if customerOrganizationID == nil {
		customerCondition = "customer_organization_id IS NULL"
	}

	rows, err := db.Query(
		ctx,
		`SELECT * FROM DeploymentTargetNotes
		WHERE deployment_target_id = @deploymentTargetID
		AND `+customerCondition,
		pgx.NamedArgs{
			"deploymentTargetID":     deploymentTargetID,
			"customerOrganizationID": customerOrganizationID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get DeploymentTargetNotes: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.DeploymentTargetNotes])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, distrerr.ErrNotFound
		}
		return nil, fmt.Errorf("failed to collect DeploymentTargetNotes: %w", err)
	}

	return &result, nil
}
