// Package logstore provides the storage backend for deployment and deployment target
// log records. The production implementation is backed by Grafana Loki; an in-memory
// fake is available for tests.
//
// All time-window resolution (defaulting "after" to the subscription's log query window
// and rejecting values older than that) and order-direction resolution happens in the
// HTTP handlers. Store implementations receive fully resolved query parameters.
package logstore

import (
	"context"
	"errors"
	"iter"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// ErrRateLimitExceeded is returned by save operations when the log store throttles the
// write (e.g. Loki's per-tenant ingestion rate limit). Callers may retry after a delay.
var ErrRateLimitExceeded = errors.New("log store rate limit exceeded")

// DeploymentLogQuery describes a read of deployment log records.
// Start and End are both inclusive, mirroring the previous SQL BETWEEN semantics.
// A zero End means "now". An empty Resources list matches all resources.
type DeploymentLogQuery struct {
	DeploymentID uuid.UUID
	Resources    []string
	Start        time.Time
	End          time.Time
	// Filter is an optional RE2 body filter, already validated by the handler.
	Filter string
	Limit  int
	// Direction is the effective order direction, resolved by the handler from the
	// client-supplied "after" parameter (before any window defaulting is applied).
	Direction types.OrderDirection
}

// DeploymentLogResourcesQuery describes a lookup of all log resources of a deployment.
// Resources with log records in one of LatestRevisionIDs are considered active, all
// others archived.
type DeploymentLogResourcesQuery struct {
	DeploymentID      uuid.UUID
	LatestRevisionIDs []uuid.UUID
	Start             time.Time
}

// DeploymentTargetLogQuery describes a read of deployment target log records.
// Start and End are both inclusive. A zero End means "now".
type DeploymentTargetLogQuery struct {
	DeploymentTargetID uuid.UUID
	Start              time.Time
	End                time.Time
	// Filter is an optional RE2 body filter, already validated by the handler.
	Filter string
	Limit  int
	// Direction is the effective order direction, resolved by the handler from the
	// client-supplied "after" parameter (before any window defaulting is applied).
	Direction types.OrderDirection
}

// LogStore stores and retrieves deployment and deployment target log records.
// The organization ID is passed explicitly on every method and maps to the Loki tenant
// (X-Scope-OrgID header) in the production implementation.
//
// Query results are streamed lazily; use util.SeqCollect to gather them into a slice.
type LogStore interface {
	SaveDeploymentLogRecords(ctx context.Context, orgID uuid.UUID, records []api.DeploymentLogRecord) error
	QueryDeploymentLogRecords(
		ctx context.Context, orgID uuid.UUID, query DeploymentLogQuery,
	) iter.Seq2[DeploymentLogRecord, error]
	GetDeploymentLogRecordResources(
		ctx context.Context, orgID uuid.UUID, query DeploymentLogResourcesQuery,
	) (active []string, archived []string, err error)

	SaveDeploymentTargetLogRecords(
		ctx context.Context, orgID, deploymentTargetID uuid.UUID, records []api.DeploymentTargetLogRecordRequest,
	) error
	QueryDeploymentTargetLogRecords(
		ctx context.Context, orgID uuid.UUID, query DeploymentTargetLogQuery,
	) iter.Seq2[DeploymentTargetLogRecord, error]
}
