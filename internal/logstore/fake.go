package logstore

import (
	"cmp"
	"context"
	"iter"
	"regexp"
	"slices"
	"sync"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// Fake is an in-memory LogStore implementation for tests.
type Fake struct {
	mu                sync.Mutex
	deploymentRecords map[uuid.UUID][]DeploymentLogRecord
	targetRecords     map[uuid.UUID][]DeploymentTargetLogRecord
}

var _ LogStore = &Fake{}

func NewFake() *Fake {
	return &Fake{
		deploymentRecords: map[uuid.UUID][]DeploymentLogRecord{},
		targetRecords:     map[uuid.UUID][]DeploymentTargetLogRecord{},
	}
}

// SaveDeploymentLogRecords implements LogStore.
func (f *Fake) SaveDeploymentLogRecords(
	_ context.Context,
	orgID uuid.UUID,
	records []api.DeploymentLogRecord,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, record := range records {
		stored := DeploymentLogRecord{
			DeploymentID:         record.DeploymentID,
			DeploymentRevisionID: record.DeploymentRevisionID,
			Resource:             record.Resource,
			Timestamp:            record.Timestamp,
			Severity:             record.Severity,
			Body:                 record.Body,
		}
		stored.ID = DeploymentLogRecordID(stored)
		f.deploymentRecords[orgID] = append(f.deploymentRecords[orgID], stored)
	}
	return nil
}

// SaveDeploymentTargetLogRecords implements LogStore.
func (f *Fake) SaveDeploymentTargetLogRecords(
	_ context.Context,
	orgID, deploymentTargetID uuid.UUID,
	records []api.DeploymentTargetLogRecordRequest,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, record := range records {
		stored := DeploymentTargetLogRecord{
			DeploymentTargetID: deploymentTargetID,
			Timestamp:          record.Timestamp,
			Severity:           record.Severity,
			Body:               record.Body,
		}
		stored.ID = DeploymentTargetLogRecordID(stored)
		f.targetRecords[orgID] = append(f.targetRecords[orgID], stored)
	}
	return nil
}

// QueryDeploymentLogRecords implements LogStore.
func (f *Fake) QueryDeploymentLogRecords(
	_ context.Context,
	orgID uuid.UUID,
	query DeploymentLogQuery,
) iter.Seq2[DeploymentLogRecord, error] {
	f.mu.Lock()
	defer f.mu.Unlock()
	filter, err := compileFilter(query.Filter)
	if err != nil {
		return seqOf[DeploymentLogRecord](nil, err)
	}
	var result []DeploymentLogRecord
	for _, record := range f.deploymentRecords[orgID] {
		if record.DeploymentID == query.DeploymentID &&
			(len(query.Resources) == 0 || slices.Contains(query.Resources, record.Resource)) &&
			inRange(record.Timestamp, query.Start, query.End) &&
			(filter == nil || filter.MatchString(record.Body)) {
			result = append(result, record)
		}
	}
	sortRecords(result, func(r DeploymentLogRecord) int64 { return r.Timestamp.UnixNano() }, query.Direction)
	return seqOf(truncate(result, query.Limit), nil)
}

// QueryDeploymentTargetLogRecords implements LogStore.
func (f *Fake) QueryDeploymentTargetLogRecords(
	_ context.Context,
	orgID uuid.UUID,
	query DeploymentTargetLogQuery,
) iter.Seq2[DeploymentTargetLogRecord, error] {
	f.mu.Lock()
	defer f.mu.Unlock()
	filter, err := compileFilter(query.Filter)
	if err != nil {
		return seqOf[DeploymentTargetLogRecord](nil, err)
	}
	var result []DeploymentTargetLogRecord
	for _, record := range f.targetRecords[orgID] {
		if record.DeploymentTargetID == query.DeploymentTargetID &&
			inRange(record.Timestamp, query.Start, query.End) &&
			(filter == nil || filter.MatchString(record.Body)) {
			result = append(result, record)
		}
	}
	sortRecords(result, func(r DeploymentTargetLogRecord) int64 { return r.Timestamp.UnixNano() }, query.Direction)
	return seqOf(truncate(result, query.Limit), nil)
}

// GetDeploymentLogRecordResources implements LogStore.
func (f *Fake) GetDeploymentLogRecordResources(
	_ context.Context,
	orgID uuid.UUID,
	query DeploymentLogResourcesQuery,
) (active []string, archived []string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	activeSet := map[string]struct{}{}
	allSet := map[string]struct{}{}
	for _, record := range f.deploymentRecords[orgID] {
		if record.DeploymentID != query.DeploymentID || record.Timestamp.Before(query.Start) {
			continue
		}
		allSet[record.Resource] = struct{}{}
		if slices.Contains(query.LatestRevisionIDs, record.DeploymentRevisionID) {
			activeSet[record.Resource] = struct{}{}
		}
	}
	for resource := range allSet {
		if _, ok := activeSet[resource]; ok {
			active = append(active, resource)
		} else {
			archived = append(archived, resource)
		}
	}
	slices.Sort(active)
	slices.Sort(archived)
	return active, archived, nil
}

func compileFilter(filter string) (*regexp.Regexp, error) {
	if filter == "" {
		return nil, nil
	}
	re, err := regexp.Compile(filter)
	if err != nil {
		return nil, apierrors.NewBadRequest("invalid filter regex")
	}
	return re, nil
}

func inRange(t, start, end time.Time) bool {
	return !t.Before(start) && (end.IsZero() || !t.After(end))
}

func sortRecords[T any](records []T, timestamp func(T) int64, direction types.OrderDirection) {
	slices.SortStableFunc(records, func(a, b T) int {
		if direction == types.OrderDirectionAsc {
			return cmp.Compare(timestamp(a), timestamp(b))
		}
		return cmp.Compare(timestamp(b), timestamp(a))
	})
}

func truncate[T any](records []T, limit int) []T {
	if limit > 0 && len(records) > limit {
		return records[:limit]
	}
	return records
}

func seqOf[T any](records []T, err error) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		if err != nil {
			yield(zero, err)
			return
		}
		for _, record := range records {
			if !yield(record, nil) {
				return
			}
		}
	}
}
