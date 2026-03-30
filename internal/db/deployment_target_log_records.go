package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/api"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const deploymentTargetLogRecordOutputExpr = `id, created_at, deployment_target_id, timestamp, severity, body`

func GetDeploymentTargetLogRecords(
	ctx context.Context,
	deploymentTargetID uuid.UUID,
	limit int,
	before, after time.Time,
	filter string,
) ([]types.DeploymentTargetLogRecord, error) {
	if before.IsZero() {
		before = time.Now()
	}

	db := internalctx.GetDb(ctx)

	filterExpr := ""
	if filter != "" {
		filterExpr = "AND body ~ @filter"
	}
	rows, err := db.Query(
		ctx,
		`SELECT `+deploymentTargetLogRecordOutputExpr+`
		FROM DeploymentTargetLogRecord
		WHERE deployment_target_id = @deployment_target_id
			AND timestamp BETWEEN @after AND @before
			`+filterExpr+`
		ORDER BY timestamp DESC
		LIMIT @limit`,
		pgx.NamedArgs{
			"deployment_target_id": deploymentTargetID,
			"after":                after,
			"before":               before,
			"limit":                limit,
			"filter":               filter,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query DeploymentTargetLogRecord: %w", err)
	}

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentTargetLogRecord])
	if err != nil {
		return nil, fmt.Errorf("could not collect DeploymentTargetLogRecord: %w", err)
	}

	return records, nil
}

func GetDeploymentTargetLogRecordsSeq(
	ctx context.Context,
	deploymentTargetID uuid.UUID,
	limit int,
) (SeqE[types.DeploymentTargetLogRecord], error) {
	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT `+deploymentTargetLogRecordOutputExpr+`
		FROM DeploymentTargetLogRecord
		WHERE deployment_target_id = @deployment_target_id
		ORDER BY timestamp DESC
		LIMIT @limit`,
		pgx.NamedArgs{
			"deployment_target_id": deploymentTargetID,
			"limit":                limit,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query DeploymentTargetLogRecord: %w", err)
	}

	return CollectSeq(rows, pgx.RowToStructByName[types.DeploymentTargetLogRecord]), nil
}

func SaveDeploymentTargetLogRecords(
	ctx context.Context,
	deploymentTargetID uuid.UUID,
	records []api.DeploymentTargetLogRecordRequest,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.CopyFrom(
		ctx,
		pgx.Identifier{"deploymenttargetlogrecord"},
		[]string{"deployment_target_id", "timestamp", "severity", "body"},
		pgx.CopyFromSlice(len(records), func(i int) ([]any, error) {
			return []any{deploymentTargetID, records[i].Timestamp, records[i].Severity, records[i].Body}, nil
		}),
	)

	return err
}

func CleanupDeploymentTargetLogRecords(ctx context.Context) (int64, error) {
	limit := env.LogRecordEntriesMaxCount()
	if limit == nil {
		return 0, nil
	}

	db := internalctx.GetDb(ctx)
	log := internalctx.GetLogger(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT `+deploymentTargetLogRecordOutputExpr+` FROM (
			SELECT *,
				row_number() OVER (PARTITION BY (deployment_target_id) ORDER BY timestamp DESC) AS rnk
				FROM DeploymentTargetLogRecord
		) lr
		WHERE rnk = @limit`,
		// "limit + 1" because we want to get the newest record that should be deleted.
		// This is an optimization to avoid unnecessary queries for resources that have exactly [limit] entries.
		pgx.NamedArgs{"limit": *limit + 1},
	)
	if err != nil {
		return 0, fmt.Errorf("error querying DeploymentTargetLogRecord: %w", err)
	}

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentTargetLogRecord])
	if err != nil {
		return 0, fmt.Errorf("error collecting DeploymentTargetLogRecord: %w", err)
	}

	var deleted int64
	var aggErr error
	for _, record := range records {
		cmd, err := db.Exec(
			ctx,
			`DELETE FROM DeploymentTargetLogRecord
			WHERE deployment_target_id = @deploymentTargetId
				AND timestamp <= @timestamp`,
			pgx.NamedArgs{"deploymentTargetId": record.DeploymentTargetID, "timestamp": record.Timestamp},
		)
		log.Debug("deleted DeploymentTargetLogRecord",
			zap.Stringer("deploymentTargetId", record.DeploymentTargetID),
			zap.Time("pivotTimestamp", record.Timestamp),
			zap.Int64("deleted", cmd.RowsAffected()),
			zap.Error(err))
		if err != nil {
			aggErr = errors.Join(aggErr, err)
		} else {
			deleted += cmd.RowsAffected()
		}
	}

	return deleted, aggErr
}
