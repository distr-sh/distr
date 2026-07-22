package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ValidateDeploymentLogRecords checks that every (deployment, revision) tuple referenced
// by the given records exists and belongs to the given deployment target. Log records
// themselves are stored in the log store (Loki), so this is the only remaining Postgres
// consistency check for pushed deployment log records.
func ValidateDeploymentLogRecords(
	ctx context.Context,
	deploymentTargetID uuid.UUID,
	records []api.DeploymentLogRecord,
) error {
	if len(records) == 0 {
		return nil
	}

	db := internalctx.GetDb(ctx)

	tuples := map[struct{ deploymentID, revisionID uuid.UUID }]struct{}{}
	for _, record := range records {
		tuples[struct{ deploymentID, revisionID uuid.UUID }{
			deploymentID: record.DeploymentID,
			revisionID:   record.DeploymentRevisionID,
		}] = struct{}{}
	}

	for tuple := range tuples {
		rows, err := db.Query(
			ctx,
			`SELECT 1
			FROM Deployment d
			JOIN DeploymentRevision dr ON d.id = dr.deployment_id
			WHERE d.deployment_target_id = @deploymentTargetId
				AND d.id = @deploymentId
				AND dr.id = @deploymentRevisionId`,
			pgx.NamedArgs{
				"deploymentTargetId":   deploymentTargetID,
				"deploymentId":         tuple.deploymentID,
				"deploymentRevisionId": tuple.revisionID,
			},
		)
		if err != nil {
			return fmt.Errorf("could not query DeploymentTarget: %w", err)
		}
		if _, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[int64]); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: deployment %s and revision %s does not exist in deployment target %s",
					apierrors.ErrNotFound, tuple.deploymentID, tuple.revisionID, deploymentTargetID)
			}
			return fmt.Errorf("could not collect DeploymentTarget: %w", err)
		}
	}

	return nil
}
