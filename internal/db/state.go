package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type OpenTofuState struct {
	ID             uuid.UUID  `db:"id"`
	DeploymentID   uuid.UUID  `db:"deployment_id"`
	OrganizationID uuid.UUID  `db:"organization_id"`
	S3Key          string     `db:"s3_key"`
	LockID         *string    `db:"lock_id"`
	LockInfo       *string    `db:"lock_info"`
	LockedAt       *time.Time `db:"locked_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
	CreatedAt      time.Time  `db:"created_at"`
}

func GetOrCreateState(ctx context.Context, deploymentID, organizationID uuid.UUID) (*OpenTofuState, error) {
	db := internalctx.GetDb(ctx)
	s3Key := fmt.Sprintf("state/%s", deploymentID.String())

	rows, err := db.Query(ctx,
		`INSERT INTO opentofu_state (deployment_id, organization_id, s3_key)
		VALUES (@deploymentID, @organizationID, @s3Key)
		ON CONFLICT (deployment_id) DO UPDATE SET updated_at = now()
		RETURNING id, deployment_id, organization_id, s3_key, lock_id, lock_info, locked_at, updated_at, created_at`,
		pgx.NamedArgs{
			"deploymentID":   deploymentID,
			"organizationID": organizationID,
			"s3Key":          s3Key,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert opentofu_state: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[OpenTofuState])
	if err != nil {
		return nil, fmt.Errorf("failed to scan opentofu_state: %w", err)
	}
	return &result, nil
}

func GetState(ctx context.Context, deploymentID uuid.UUID) (*OpenTofuState, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(ctx,
		`SELECT id, deployment_id, organization_id, s3_key, lock_id, lock_info, locked_at, updated_at, created_at
		FROM opentofu_state
		WHERE deployment_id = @deploymentID`,
		pgx.NamedArgs{"deploymentID": deploymentID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query opentofu_state: %w", err)
	}

	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[OpenTofuState])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apierrors.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get opentofu_state: %w", err)
	}
	return &result, nil
}

func LockState(ctx context.Context, deploymentID uuid.UUID, lockID, lockInfo string) error {
	db := internalctx.GetDb(ctx)

	cmd, err := db.Exec(ctx,
		`UPDATE opentofu_state
		SET lock_id = @lockID, lock_info = @lockInfo, locked_at = now(), updated_at = now()
		WHERE deployment_id = @deploymentID
			AND (lock_id IS NULL OR lock_id = @lockID OR locked_at < now() - interval '1 hour')`,
		pgx.NamedArgs{
			"deploymentID": deploymentID,
			"lockID":       lockID,
			"lockInfo":     lockInfo,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to lock opentofu_state: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apierrors.ErrConflict
	}
	return nil
}

func UnlockState(ctx context.Context, deploymentID uuid.UUID, lockID string) error {
	db := internalctx.GetDb(ctx)

	cmd, err := db.Exec(ctx,
		`UPDATE opentofu_state
		SET lock_id = NULL, lock_info = NULL, locked_at = NULL, updated_at = now()
		WHERE deployment_id = @deploymentID
			AND lock_id = @lockID`,
		pgx.NamedArgs{
			"deploymentID": deploymentID,
			"lockID":       lockID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to unlock opentofu_state: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apierrors.ErrConflict
	}
	return nil
}
