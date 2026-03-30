package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

const (
	deploymentLogRecordOutputExpr = `
	lr.id, lr.created_at, lr.deployment_id, lr.deployment_revision_id, lr.resource, lr.timestamp, lr.severity, lr.body
	`
)

func SaveDeploymentLogRecords(ctx context.Context, records []api.DeploymentLogRecord) error {
	db := internalctx.GetDb(ctx)
	_, err := db.CopyFrom(
		ctx,
		pgx.Identifier{"deploymentlogrecord"},
		[]string{"deployment_id", "deployment_revision_id", "resource", "timestamp", "severity", "body"},
		pgx.CopyFromSlice(len(records), func(i int) ([]any, error) {
			r := records[i]
			return []any{r.DeploymentID, r.DeploymentRevisionID, r.Resource, r.Timestamp, r.Severity, r.Body}, nil
		}),
	)
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		if pgerr.Code == pgerrcode.ForeignKeyViolation {
			return apierrors.NewBadRequest("deployment does not exist")
		}
	}
	return err
}

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

func GetDeploymentLogRecordResources(ctx context.Context,
	deploymentID uuid.UUID,
) (active []string, archived []string, err error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH latest_revisions AS (
			SELECT id FROM DeploymentRevision
			WHERE deployment_id = @deploymentId
			ORDER BY created_at DESC
			LIMIT 5
		),
		active_resources AS (
			SELECT DISTINCT resource
			FROM DeploymentLogRecord
			WHERE deployment_id = @deploymentId
				AND deployment_revision_id IN (SELECT id FROM latest_revisions)
		),
		all_resources AS (
			SELECT DISTINCT resource
			FROM DeploymentLogRecord
			WHERE deployment_id = @deploymentId
		)
		SELECT resource,
			CASE WHEN ar.resource IS NOT NULL THEN false ELSE true END AS is_archived
		FROM all_resources all_r
		LEFT JOIN active_resources ar USING (resource)
		ORDER BY resource`,
		pgx.NamedArgs{"deploymentId": deploymentID},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not query DeploymentLogRecord: %w", err)
	}

	type resourceRow struct {
		Resource   string
		IsArchived bool
	}
	results, err := pgx.CollectRows(rows, pgx.RowToStructByPos[resourceRow])
	if err != nil {
		return nil, nil, fmt.Errorf("could not collect DeploymentLogRecord: %w", err)
	}

	for _, r := range results {
		if r.IsArchived {
			archived = append(archived, r.Resource)
		} else {
			active = append(active, r.Resource)
		}
	}
	return active, archived, nil
}

func GetDeploymentLogRecords(
	ctx context.Context,
	deploymentID uuid.UUID,
	resource string,
	limit int,
	before time.Time,
	after time.Time,
	filter string,
) ([]types.DeploymentLogRecord, error) {
	if before.IsZero() {
		before = time.Now()
	}
	db := internalctx.GetDb(ctx)
	filterExpr := ""
	if filter != "" {
		filterExpr = "AND lr.body ~ @filter"
	}
	rows, err := db.Query(
		ctx,
		`SELECT `+deploymentLogRecordOutputExpr+`
		FROM DeploymentLogRecord lr
		WHERE lr.deployment_id = @deploymentId
			AND lr.resource = @resource
			AND lr.timestamp BETWEEN @after AND @before
			`+filterExpr+`
		ORDER BY lr.timestamp DESC
		LIMIT @limit`,
		pgx.NamedArgs{
			"deploymentId": deploymentID,
			"resource":     resource,
			"limit":        limit,
			"before":       before,
			"after":        after,
			"filter":       filter,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query DeploymentLogRecord: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentLogRecord])
	if err != nil {
		return nil, fmt.Errorf("could not collect DeploymentLogRecord: %w", err)
	}
	return result, nil
}

// GetDeploymentLogRecordsForExport retrieves deployment log records for export
// ordered by timestamp DESC with a subscription-based limit.
// The callback function is called for each row, allowing true streaming without loading all rows into memory.
func GetDeploymentLogRecordsForExport(
	ctx context.Context,
	deploymentID uuid.UUID,
	resource string,
	limit int,
	callback func(types.DeploymentLogRecord) error,
) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT `+deploymentLogRecordOutputExpr+`
		FROM DeploymentLogRecord lr
		WHERE lr.deployment_id = @deploymentId
			AND lr.resource = @resource
		ORDER BY lr.timestamp DESC
		LIMIT @limit`,
		pgx.NamedArgs{
			"deploymentId": deploymentID,
			"resource":     resource,
			"limit":        limit,
		},
	)
	if err != nil {
		return fmt.Errorf("could not query DeploymentLogRecord: %w", err)
	}

	var record types.DeploymentLogRecord
	_, err = pgx.ForEachRow(rows, []any{
		&record.ID,
		&record.CreatedAt,
		&record.DeploymentID,
		&record.DeploymentRevisionID,
		&record.Resource,
		&record.Timestamp,
		&record.Severity,
		&record.Body,
	}, func() error {
		return callback(record)
	})
	if err != nil {
		return fmt.Errorf("could not iterate DeploymentLogRecord: %w", err)
	}

	return nil
}

func BulkCreateDeploymentLogRecordWithCreatedAt(
	ctx context.Context,
	deploymentID uuid.UUID,
	deploymentRevisionID uuid.UUID,
	records []types.DeploymentLogRecord,
) error {
	db := internalctx.GetDb(ctx)
	_, err := db.CopyFrom(
		ctx,
		pgx.Identifier{"deploymentlogrecord"},
		[]string{"deployment_id", "deployment_revision_id", "resource", "created_at", "timestamp", "severity", "body"},
		pgx.CopyFromSlice(len(records), func(i int) ([]any, error) {
			return []any{
				deploymentID,
				deploymentRevisionID,
				records[i].Resource,
				records[i].CreatedAt,
				records[i].Timestamp,
				records[i].Severity,
				records[i].Body,
			}, nil
		}),
	)
	return err
}

// CleanupDeploymentLogRecords deletes logrecords for all deployments but keeps the
// last [env.LogRecordEntriesMaxCount] records for each (deployment_id, resource) group.
//
// If [env.LogRecordEntriesMaxCount] is nil, no cleanup is performed.
func CleanupDeploymentLogRecords(ctx context.Context) (int64, error) {
	limit := env.LogRecordEntriesMaxCount()
	if limit == nil {
		return 0, nil
	}

	db := internalctx.GetDb(ctx)
	log := internalctx.GetLogger(ctx)

	rows, err := db.Query(
		ctx,
		`SELECT `+deploymentLogRecordOutputExpr+` FROM (
			SELECT *,
				row_number() OVER (PARTITION BY (deployment_id, resource) ORDER BY timestamp DESC) AS rnk
				FROM DeploymentLogRecord
		) lr
		WHERE rnk = @limit`,
		// "limit + 1" because we want to get the newest record that should be deleted.
		// This is an optimization to avoid unnecessary queries for resources that have exactly [limit] entries.
		pgx.NamedArgs{"limit": *limit + 1},
	)
	if err != nil {
		return 0, fmt.Errorf("error querying DeploymentLogRecords: %w", err)
	}

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.DeploymentLogRecord])
	if err != nil {
		return 0, fmt.Errorf("error collecting rows: %w", err)
	}

	var deleted int64
	var aggErr error
	for _, record := range records {
		cmd, err := db.Exec(
			ctx,
			`DELETE FROM DeploymentLogRecord
			WHERE deployment_id = @deploymentId
				AND resource = @resource
				AND timestamp <= @timestamp`,
			pgx.NamedArgs{"deploymentId": record.DeploymentID, "resource": record.Resource, "timestamp": record.Timestamp},
		)
		log.Debug("deleted DeploymentLogRecords",
			zap.Stringer("deploymentId", record.DeploymentID),
			zap.String("resource", record.Resource),
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
