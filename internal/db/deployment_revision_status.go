package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// UpdateDeploymentRevisionStatus overwrites the status of the given revision. Agents report the
// status they observe on every interval, so created_at is bumped even when nothing changed, which
// is what [types.DeploymentRevisionStatus.IsStale] reads to tell a silent agent from a healthy one.
// Any status other than progressing also becomes the revision's settled status, which a progressing
// status leaves alone (see [GetPreviousDeploymentRevisionStatus]).
func UpdateDeploymentRevisionStatus(
	ctx context.Context,
	deploymentRevisionID uuid.UUID,
	statusType types.DeploymentStatusType,
	message string,
) (*types.DeploymentRevisionStatus, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`UPDATE DeploymentRevision SET
			status_type = @type::DEPLOYMENT_STATUS_TYPE,
			status_message = @message,
			status_created_at = now(),
			settled_status_type = CASE WHEN @type::DEPLOYMENT_STATUS_TYPE = 'progressing'
				THEN settled_status_type ELSE @type::DEPLOYMENT_STATUS_TYPE END,
			settled_status_created_at = CASE WHEN @type::DEPLOYMENT_STATUS_TYPE = 'progressing'
				THEN settled_status_created_at ELSE now() END
		WHERE id = @deploymentRevisionId
		RETURNING id AS deployment_revision_id, status_created_at AS created_at, status_type AS type,
			status_message AS message`,
		pgx.NamedArgs{
			"deploymentRevisionId": deploymentRevisionID,
			"type":                 statusType,
			"message":              message,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update DeploymentRevision status: %w", err)
	}

	status, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.DeploymentRevisionStatus])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: unknown deployment revision", apierrors.ErrConflict)
	} else if err != nil {
		return nil, fmt.Errorf("failed to collect DeploymentRevision status: %w", err)
	}

	RunAfterTx(ctx, func(ctx context.Context) {
		log := internalctx.GetLogger(ctx)
		if c := internalctx.GetPrometheusCollector(ctx); c != nil {
			if m, err := GetDeploymentForMetricsByRevisionID(ctx, deploymentRevisionID); err != nil {
				log.Error("could not update deployment status metrics", zap.Error(err))
			} else {
				c.HandleDeploymentStatus(*m)
			}
		} else {
			log.Warn("could not update deployment status metrics because collector is nil")
		}
	})

	return &status, nil
}

// GetPreviousDeploymentRevisionStatus returns the status that a new status of the given revision is compared
// with to decide about notifications: the settled status of the same revision or, if it has none yet, the
// settled status of the deployment's current revision. It never compares with the newest status across all
// revisions, because agents report the current and the latest revision independently, so that status flips
// between the two. It ignores progressing, because an agent that retries applying a revision reports it
// between two errors, which would otherwise alert on every retry. The message of a settled status is not
// stored.
func GetPreviousDeploymentRevisionStatus(
	ctx context.Context,
	revisionID uuid.UUID,
) (*types.DeploymentRevisionStatus, error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT dr.id AS deployment_revision_id, dr.settled_status_created_at AS created_at,
			dr.settled_status_type AS type, '' AS message
		FROM DeploymentRevision dr
		WHERE dr.settled_status_type IS NOT NULL
			AND dr.id IN (
				@revisionId,
				(
					SELECT d.current_deployment_revision_id
					FROM Deployment d
						JOIN DeploymentRevision r ON r.deployment_id = d.id
					WHERE r.id = @revisionId
				)
			)
		ORDER BY dr.id = @revisionId DESC
		LIMIT 1`,
		pgx.NamedArgs{"revisionId": revisionID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query previous DeploymentRevision status: %w", err)
	}

	if result, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.DeploymentRevisionStatus]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to collect DeploymentRevision status: %w", err)
	} else {
		return &result, nil
	}
}
