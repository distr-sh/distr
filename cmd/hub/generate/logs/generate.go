package main

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/logstore"
	"github.com/distr-sh/distr/internal/svc"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
)

func main() {
	ctx := context.Background()
	env.Initialize()
	registry := util.Require(svc.NewDefault(ctx))
	defer func() { _ = registry.Shutdown(ctx) }()
	logStore := registry.GetLogStore()

	orgID := uuid.MustParse("f720da7c-d7fa-4c7a-959b-40ebfd13703b")
	deploymentID := uuid.MustParse("b053ac4f-28eb-49cc-88a1-5debc3ec3dc1")
	revisionID := uuid.MustParse("7a70c13c-4d5c-4344-9576-ff7a9a155726")
	// The generated time span (statusCount * statusInterval, here ~5.8 days) must stay
	// below Loki's reject_old_samples max age (default 1 week), otherwise the oldest
	// samples are rejected.
	statusCount := 2_000_000
	statusInterval := 250 * time.Millisecond
	batchSize := 1000

	now := time.Now().UTC()
	timestamp := now.Add(time.Duration(statusCount) * -statusInterval)
	// Resume after the newest existing record: Loki rejects entries older than a
	// stream's newest timestamp minus the out-of-order window ("entry too far behind"),
	// so re-runs must not write into the already-covered part of the stream.
	if newest, ok := newestExistingTimestamp(ctx, logStore, orgID, deploymentID); ok && newest.After(timestamp) {
		timestamp = newest.Add(statusInterval)
	}
	batch := make([]api.DeploymentLogRecord, 0, batchSize)
	for timestamp.Before(now) {
		batch = append(batch, api.DeploymentLogRecord{
			DeploymentID:         deploymentID,
			DeploymentRevisionID: revisionID,
			Resource:             "example-resource",
			Timestamp:            timestamp,
			Severity:             "error",
			Body:                 randomString(1000),
		})
		if len(batch) == batchSize {
			saveWithRetry(ctx, logStore, orgID, batch)
			batch = batch[:0]
		}
		timestamp = timestamp.Add(statusInterval)
	}
	if len(batch) > 0 {
		saveWithRetry(ctx, logStore, orgID, batch)
	}
}

func newestExistingTimestamp(
	ctx context.Context,
	logStore logstore.LogStore,
	orgID, deploymentID uuid.UUID,
) (time.Time, bool) {
	// 30 days matches the shipped retention_period and stays within Loki's
	// max_query_length limit (default 30d1h).
	records := util.Require(util.SeqCollect(logStore.QueryDeploymentLogRecords(ctx, orgID, logstore.DeploymentLogQuery{
		DeploymentID: deploymentID,
		Start:        time.Now().UTC().Add(-30 * 24 * time.Hour),
		Limit:        1,
		Direction:    types.OrderDirectionDesc,
	})))
	if len(records) == 0 {
		return time.Time{}, false
	}
	return records[0].Timestamp, true
}

// saveWithRetry backs off and retries when the log store throttles the write, since the
// generator pushes far faster than real agents and easily exceeds Loki's per-tenant
// ingestion rate limit.
func saveWithRetry(ctx context.Context, logStore logstore.LogStore, orgID uuid.UUID, batch []api.DeploymentLogRecord) {
	for {
		err := logStore.SaveDeploymentLogRecords(ctx, orgID, batch)
		if errors.Is(err, logstore.ErrRateLimitExceeded) {
			time.Sleep(time.Second)
			continue
		}
		util.Must(err)
		return
	}
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 "
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
