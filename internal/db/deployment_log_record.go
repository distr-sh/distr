package db

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/api"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type deploymentRevisionTuple struct{ deploymentID, revisionID uuid.UUID }

// FilterValidDeploymentLogRecords returns the subset of records whose (deployment, revision)
// tuple exists and belongs to the given deployment target. Records referencing unknown tuples
// are dropped instead of failing the whole batch, so valid records from other deployments in
// the same batch are still persisted. Log records themselves are stored in the log store (Loki),
// so this is the only remaining Postgres consistency check for pushed deployment log records.
func FilterValidDeploymentLogRecords(
	ctx context.Context,
	deploymentTargetID uuid.UUID,
	records []api.DeploymentLogRecord,
) ([]api.DeploymentLogRecord, error) {
	if len(records) == 0 {
		return records, nil
	}

	db := internalctx.GetDb(ctx)

	tuples := map[deploymentRevisionTuple]struct{}{}
	for _, record := range records {
		tuples[deploymentRevisionTuple{record.DeploymentID, record.DeploymentRevisionID}] = struct{}{}
	}

	deploymentIDs := make([]uuid.UUID, 0, len(tuples))
	revisionIDs := make([]uuid.UUID, 0, len(tuples))
	for tuple := range tuples {
		deploymentIDs = append(deploymentIDs, tuple.deploymentID)
		revisionIDs = append(revisionIDs, tuple.revisionID)
	}

	rows, err := db.Query(
		ctx,
		`SELECT d.id, dr.id
		FROM Deployment d
		JOIN DeploymentRevision dr ON d.id = dr.deployment_id
		WHERE d.deployment_target_id = @deploymentTargetId
			AND (d.id, dr.id) IN (SELECT * FROM unnest(@deploymentIds::uuid[], @revisionIds::uuid[]))`,
		pgx.NamedArgs{
			"deploymentTargetId": deploymentTargetID,
			"deploymentIds":      deploymentIDs,
			"revisionIds":        revisionIDs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query deployment revisions: %w", err)
	}
	validTuples, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (deploymentRevisionTuple, error) {
		var tuple deploymentRevisionTuple
		err := row.Scan(&tuple.deploymentID, &tuple.revisionID)
		return tuple, err
	})
	if err != nil {
		return nil, fmt.Errorf("could not collect deployment revisions: %w", err)
	}

	valid := make(map[deploymentRevisionTuple]struct{}, len(validTuples))
	for _, tuple := range validTuples {
		valid[tuple] = struct{}{}
	}

	filtered := make([]api.DeploymentLogRecord, 0, len(records))
	for _, record := range records {
		if _, ok := valid[deploymentRevisionTuple{record.DeploymentID, record.DeploymentRevisionID}]; ok {
			filtered = append(filtered, record)
		}
	}

	return filtered, nil
}
