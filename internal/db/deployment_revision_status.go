package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func CreateDeploymentRevisionStatus(ctx context.Context, status *types.DeploymentRevisionStatus) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH inserted AS (
			INSERT INTO DeploymentRevisionStatus (deployment_revision_id, message, type)
			VALUES (@deploymentRevisionId, @message, @type)
			RETURNING *
		)
		SELECT id, created_at, deployment_revision_id, type, message FROM inserted
		`,
		pgx.NamedArgs{
			"deploymentRevisionId": status.DeploymentRevisionID,
			"message":              status.Message,
			"type":                 status.Type,
		},
	)
	if err != nil {
		return err
	}

	if res, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.DeploymentRevisionStatus]); err != nil {
		if pgErr := new(pgconn.PgError); errors.As(err, &pgErr) && pgErr.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*status = res
	}

	return nil
}

func BulkCreateDeploymentRevisionStatusWithCreatedAt(
	ctx context.Context,
	deploymentRevisionID uuid.UUID,
	statuses []types.DeploymentRevisionStatus,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.CopyFrom(
		ctx,
		pgx.Identifier{"deploymentrevisionstatus"},
		[]string{"deployment_revision_id", "type", "message", "created_at"},
		pgx.CopyFromSlice(len(statuses), func(i int) ([]any, error) {
			return []any{
				deploymentRevisionID,
				types.DeploymentStatusTypeHealthy,
				statuses[i].Message,
				statuses[i].CreatedAt,
			}, nil
		}),
	)
	return err
}

func GetDeploymentRevisionStatus(
	ctx context.Context,
	deploymentID uuid.UUID,
	maxRows int,
	before time.Time,
	after time.Time,
) ([]types.DeploymentRevisionStatus, error) {
	if before.IsZero() {
		before = time.Now()
	}

	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		"SELECT id from DeploymentRevision WHERE deployment_id = @deploymentId",
		pgx.NamedArgs{"deploymentId": deploymentID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentRevision for status: %w", err)
	}
	deploymentRevisionIDs, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("failed to scan DeploymentRevision for status: %w", err)
	}

	rows, err = db.Query(
		ctx,
		`SELECT id, created_at, deployment_revision_id, type, message
		FROM DeploymentRevisionStatus
		WHERE deployment_revision_id = ANY (@deploymentRevisionIds)
			AND created_at BETWEEN @after AND @before
		ORDER BY created_at DESC
		LIMIT @maxRows`,
		pgx.NamedArgs{
			"deploymentRevisionIds": deploymentRevisionIDs,
			"maxRows":               maxRows,
			"before":                before,
			"after":                 after,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query DeploymentRevisionStatus: %w", err)
	} else if result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentRevisionStatus]); err != nil {
		return nil, fmt.Errorf("failed to get DeploymentRevisionStatus: %w", err)
	} else {
		return result, nil
	}
}

func GetDeploymentRevisionStatusForExport(
	ctx context.Context,
	deploymentID uuid.UUID,
	limit int,
	callback func(types.DeploymentRevisionStatus) error,
) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		"SELECT id from DeploymentRevision WHERE deployment_id = @deploymentId",
		pgx.NamedArgs{"deploymentId": deploymentID},
	)
	if err != nil {
		return fmt.Errorf("failed to query DeploymentRevision for status: %w", err)
	}
	deploymentRevisionIDs, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return fmt.Errorf("failed to scan DeploymentRevision for status: %w", err)
	}

	rows, err = db.Query(
		ctx,
		`SELECT id, created_at, deployment_revision_id, type, message
		FROM DeploymentRevisionStatus
		WHERE deployment_revision_id = ANY (@deploymentRevisionIds)
		ORDER BY created_at DESC
		LIMIT @limit`,
		pgx.NamedArgs{
			"deploymentRevisionIds": deploymentRevisionIDs,
			"limit":                 limit,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to query DeploymentRevisionStatus: %w", err)
	}

	var status types.DeploymentRevisionStatus
	_, err = pgx.ForEachRow(rows, []any{
		&status.ID,
		&status.CreatedAt,
		&status.DeploymentRevisionID,
		&status.Type,
		&status.Message,
	}, func() error {
		return callback(status)
	})
	if err != nil {
		return fmt.Errorf("could not iterate DeploymentRevisionStatus: %w", err)
	}

	return nil
}

func GetLatestDeploymentRevisionStatus(
	ctx context.Context,
	deploymentID uuid.UUID,
) (*types.DeploymentRevisionStatus, error) {
	db := internalctx.GetDb(ctx)

	revisionIDs, err := GetDeploymentRevisionIDs(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		ctx,
		`SELECT id, created_at, deployment_revision_id, type, message
		FROM DeploymentRevisionStatus
		WHERE deployment_revision_id = ANY(@revisionIDs)
		ORDER BY created_at DESC
		LIMIT 1`,
		pgx.NamedArgs{"revisionIDs": revisionIDs},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest DeploymentRevisionStatus: %w", err)
	}

	if result, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.DeploymentRevisionStatus]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to collect DeploymentRevisionStatus: %w", err)
	} else {
		return &result, nil
	}
}

// CleanupDeploymentRevisionStatus deletes all DeploymentRevisionStatus entries older than [env.StatusEntriesMaxAge()],
// always keeping the latest entry across all DeploymentRevisions of every Deployment
func CleanupDeploymentRevisionStatus(ctx context.Context) (int64, error) {
	if env.StatusEntriesMaxAge() == nil {
		return 0, nil
	}

	db := internalctx.GetDb(ctx)
	if cmd, err := db.Exec(
		ctx,
		`DELETE FROM DeploymentRevisionStatus drs
		USING (
			SELECT
				dr1.id AS deployment_revision_id,
				max(dr2.max_created_at) AS max_created_at
			FROM DeploymentRevision dr1
			JOIN (
				SELECT dr.id, dr.deployment_id, (
					SELECT max(drs.created_at)
					FROM DeploymentRevisionStatus drs
					WHERE drs.deployment_revision_id = dr.id
				) AS max_created_at
				FROM DeploymentRevision dr
			) dr2 ON dr1.deployment_id = dr2.deployment_id
			GROUP BY dr1.id
		) max_created_at
		WHERE drs.deployment_revision_id = max_created_at.deployment_revision_id
			AND drs.created_at < max_created_at.max_created_at
			AND current_timestamp - drs.created_at > @statusEntriesMaxAge`,
		pgx.NamedArgs{"statusEntriesMaxAge": env.StatusEntriesMaxAge()},
	); err != nil {
		return 0, err
	} else {
		return cmd.RowsAffected(), nil
	}
}
