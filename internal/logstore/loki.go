package logstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	labelKind                 = "distr_kind"
	labelDeploymentID         = "deployment_id"
	labelDeploymentRevisionID = "deployment_revision_id"
	labelResource             = "resource"
	labelDeploymentTargetID   = "deployment_target_id"
	metadataSeverity          = "severity"

	kindDeployment       = "deployment"
	kindDeploymentTarget = "deployment_target"

	// defaultMaxEntriesPerQuery must not exceed Loki's
	// limits_config.max_entries_limit_per_query (default 5000). Streaming reads page
	// through larger limits in chunks of this size.
	defaultMaxEntriesPerQuery = 5000

	defaultRequestTimeout = 30 * time.Second
)

type LokiConfig struct {
	// URL is the base URL of the Loki instance, e.g. "http://loki:3100".
	URL string
	// BearerToken enables bearer token authentication when non-nil.
	BearerToken *string
	// BasicAuthUsername and BasicAuthPassword enable basic authentication when both are non-nil.
	BasicAuthUsername *string
	BasicAuthPassword *string
	// RequestTimeout is the timeout for a single Loki HTTP request. Defaults to 30s.
	RequestTimeout time.Duration
}

type lokiStore struct {
	config LokiConfig
	client *http.Client
	// maxEntriesPerQuery is the page size for streaming reads (overridable in tests).
	maxEntriesPerQuery int
}

// NewLokiStore creates the Loki-backed LogStore. The organization ID passed to each
// method is sent as the Loki tenant (X-Scope-OrgID header).
func NewLokiStore(config LokiConfig) (LogStore, error) {
	if _, err := url.Parse(config.URL); err != nil {
		return nil, fmt.Errorf("invalid loki URL: %w", err)
	}

	timeout := config.RequestTimeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	return &lokiStore{
		config:             config,
		client:             &http.Client{Timeout: timeout},
		maxEntriesPerQuery: defaultMaxEntriesPerQuery,
	}, nil
}

// SaveDeploymentLogRecords implements LogStore.
func (s *lokiStore) SaveDeploymentLogRecords(
	ctx context.Context,
	orgID uuid.UUID,
	records []api.DeploymentLogRecord,
) error {
	if len(records) == 0 {
		return nil
	}

	streams := map[string]*lokiStream{}
	for _, record := range records {
		labels := map[string]string{
			labelKind:                 kindDeployment,
			labelDeploymentID:         record.DeploymentID.String(),
			labelDeploymentRevisionID: record.DeploymentRevisionID.String(),
			labelResource:             record.Resource,
		}
		appendToStream(streams, labels, record.Timestamp, record.Body, record.Severity)
	}

	return s.push(ctx, orgID, streams)
}

// SaveDeploymentTargetLogRecords implements LogStore.
func (s *lokiStore) SaveDeploymentTargetLogRecords(
	ctx context.Context,
	orgID, deploymentTargetID uuid.UUID,
	records []api.DeploymentTargetLogRecordRequest,
) error {
	if len(records) == 0 {
		return nil
	}

	streams := map[string]*lokiStream{}
	labels := map[string]string{
		labelKind:               kindDeploymentTarget,
		labelDeploymentTargetID: deploymentTargetID.String(),
	}

	for _, record := range records {
		appendToStream(streams, labels, record.Timestamp, record.Body, record.Severity)
	}

	return s.push(ctx, orgID, streams)
}

// QueryDeploymentLogRecords implements LogStore.
func (s *lokiStore) QueryDeploymentLogRecords(
	ctx context.Context,
	orgID uuid.UUID,
	query DeploymentLogQuery,
) iter.Seq2[DeploymentLogRecord, error] {
	logql := deploymentSelector(query.DeploymentID, query.Resources) + filterExpr(query.Filter)
	return querySeq(s, ctx, orgID, logql, query.Start, query.End, query.Limit, query.Direction,
		entryToDeploymentLogRecord)
}

// QueryDeploymentTargetLogRecords implements LogStore.
func (s *lokiStore) QueryDeploymentTargetLogRecords(
	ctx context.Context,
	orgID uuid.UUID,
	query DeploymentTargetLogQuery,
) iter.Seq2[DeploymentTargetLogRecord, error] {
	logql := deploymentTargetSelector(query.DeploymentTargetID) + filterExpr(query.Filter)
	return querySeq(s, ctx, orgID, logql, query.Start, query.End, query.Limit, query.Direction,
		entryToDeploymentTargetLogRecord)
}

// GetDeploymentLogRecordResources implements LogStore.
func (s *lokiStore) GetDeploymentLogRecordResources(
	ctx context.Context,
	orgID uuid.UUID,
	query DeploymentLogResourcesQuery,
) (active []string, archived []string, err error) {
	end := time.Now()

	allResources, err := s.series(ctx, orgID, deploymentSelector(query.DeploymentID, nil), query.Start, end)
	if err != nil {
		return nil, nil, err
	}

	activeSet := map[string]struct{}{}
	if len(query.LatestRevisionIDs) > 0 {
		activeResources, err := s.series(
			ctx, orgID,
			deploymentRevisionsSelector(query.DeploymentID, query.LatestRevisionIDs),
			query.Start, end,
		)
		if err != nil {
			return nil, nil, err
		}

		for _, resource := range activeResources {
			activeSet[resource] = struct{}{}
		}
	}

	for _, resource := range allResources {
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

// querySeq lazily pages through query_range responses until limit entries have been
// produced or the window is exhausted. Loki caps a single response at
// max_entries_limit_per_query, so after each page the range bound on the read side
// advances to the last returned entry's timestamp: the end bound moves backwards for
// descending reads, the start bound forwards for ascending reads. Entries at that
// boundary timestamp that were already emitted are re-fetched and skipped by
// fingerprint. This is necessary because entries in different streams can share the
// exact same nanosecond (increment_duplicate_timestamp only de-duplicates within a
// stream).
func querySeq[T any](
	s *lokiStore,
	ctx context.Context,
	orgID uuid.UUID,
	logql string,
	start, end time.Time,
	limit int,
	direction types.OrderDirection,
	mapEntry func(lokiEntry) (T, error),
) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T

		if end.IsZero() {
			// Resolve "now" once so the window stays stable across pages.
			end = time.Now()
		}

		logger := internalctx.GetLogger(ctx)
		emittedAtBoundary := map[string]struct{}{}
		remaining := limit
		page := 0
		for remaining > 0 {
			page++
			logger.Debug("logstore query page",
				zap.Int("page", page), zap.Int("remaining", remaining),
				zap.Time("start", start), zap.Time("end", end))
			// Entries at the boundary timestamp that were already emitted are re-fetched
			// and skipped, so they must not count against the remaining budget.
			pageLimit := min(remaining+len(emittedAtBoundary), s.maxEntriesPerQuery)
			entries, err := s.queryRange(ctx, orgID, logql, start, end, pageLimit, direction)
			if err != nil {
				yield(zero, err)
				return
			}

			emitted := 0
			for _, entry := range entries {
				if _, ok := emittedAtBoundary[entry.fingerprint()]; ok {
					continue
				}

				record, err := mapEntry(entry)
				if err != nil {
					yield(zero, err)
					return
				}

				if !yield(record, nil) {
					return
				}

				emitted++
				remaining--
				if remaining == 0 {
					return
				}
			}

			if emitted == 0 || len(entries) < pageLimit {
				logger.Debug("logstore query exhausted",
					zap.Int("pages", page), zap.Int("produced", limit-remaining))
				return
			}

			// Both range bounds are inclusive in queryRange terms: passing the boundary
			// timestamp itself re-fetches entries at exactly this nanosecond on the next
			// page (skipped above by fingerprint).
			boundary := entries[len(entries)-1].timestamp
			if direction == types.OrderDirectionAsc {
				start = boundary
			} else {
				end = boundary
			}

			clear(emittedAtBoundary)
			for _, entry := range entries {
				if entry.timestamp.Equal(boundary) {
					emittedAtBoundary[entry.fingerprint()] = struct{}{}
				}
			}
		}
	}
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]any           `json:"values"`
}

func appendToStream(
	streams map[string]*lokiStream,
	labels map[string]string,
	timestamp time.Time,
	body, severity string,
) {
	key := streamKey(labels)

	stream, ok := streams[key]
	if !ok {
		stream = &lokiStream{Stream: labels}
		streams[key] = stream
	}

	value := []any{strconv.FormatInt(timestamp.UnixNano(), 10), body}

	if severity != "" {
		value = append(value, map[string]string{metadataSeverity: severity})
	}

	stream.Values = append(stream.Values, value)
}

func streamKey(labels map[string]string) string {
	var b strings.Builder

	for _, key := range slices.Sorted(maps.Keys(labels)) {
		b.WriteString(key)
		b.WriteByte(0)
		b.WriteString(labels[key])
		b.WriteByte(0)
	}

	return b.String()
}

func (s *lokiStore) push(ctx context.Context, orgID uuid.UUID, streams map[string]*lokiStream) error {
	body := struct {
		Streams []*lokiStream `json:"streams"`
	}{Streams: slices.Collect(maps.Values(streams))}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("could not marshal loki push request: %w", err)
	}

	req, err := s.newRequest(ctx, orgID, http.MethodPost, "/loki/api/v1/push", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(bytes.NewReader(data))
	req.ContentLength = int64(len(data))
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("loki push failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusBadRequest {
		return apierrors.NewBadRequest("loki rejected log records: " + readErrorBody(resp))
	} else if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("%w: %v", ErrRateLimitExceeded, readErrorBody(resp))
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("loki push failed with status %v: %v", resp.StatusCode, readErrorBody(resp))
	}

	return nil
}

// lokiEntry is a single log line returned by a query, together with the labels of the
// stream it belongs to (which include structured metadata such as severity, since Loki
// merges it into the response labels by default).
type lokiEntry struct {
	labels    map[string]string
	timestamp time.Time
	line      string
}

func (e lokiEntry) fingerprint() string {
	return streamKey(e.labels) + strconv.FormatInt(e.timestamp.UnixNano(), 10) + "\x00" + e.line
}

// queryRange runs a query_range request. Start and End are inclusive; Loki's exclusive
// end parameter is therefore sent as End+1ns. A zero End means "now".
func (s *lokiStore) queryRange(
	ctx context.Context,
	orgID uuid.UUID,
	logql string,
	start, end time.Time,
	limit int,
	direction types.OrderDirection,
) ([]lokiEntry, error) {
	if end.IsZero() {
		end = time.Now()
	}

	lokiDirection := "backward"
	if direction == types.OrderDirectionAsc {
		lokiDirection = "forward"
	}

	params := url.Values{
		"query":     []string{logql},
		"start":     []string{strconv.FormatInt(start.UnixNano(), 10)},
		"end":       []string{strconv.FormatInt(end.UnixNano()+1, 10)},
		"limit":     []string{strconv.Itoa(limit)},
		"direction": []string{lokiDirection},
	}

	logger := internalctx.GetLogger(ctx)
	logger.Debug("loki query_range request",
		zap.String("query", logql),
		zap.Time("start", start), zap.Time("end", end),
		zap.Int("limit", limit), zap.String("direction", lokiDirection))
	began := time.Now()

	req, err := s.newRequest(ctx, orgID, http.MethodGet, "/loki/api/v1/query_range", params)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("loki query failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusBadRequest {
		return nil, apierrors.NewBadRequest("loki rejected query: " + readErrorBody(resp))
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("loki query failed with status %v: %v", resp.StatusCode, readErrorBody(resp))
	}

	var body struct {
		Data struct {
			Result []struct {
				Stream map[string]string `json:"stream"`
				Values [][]string        `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("could not decode loki query response: %w", err)
	}

	var entries []lokiEntry
	for _, result := range body.Data.Result {
		for _, value := range result.Values {
			if len(value) < 2 {
				return nil, fmt.Errorf("unexpected loki value format: %v", value)
			}
			nanos, err := strconv.ParseInt(value[0], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("could not parse loki entry timestamp: %w", err)
			}
			entries = append(entries, lokiEntry{
				labels:    result.Stream,
				timestamp: time.Unix(0, nanos).UTC(),
				line:      value[1],
			})
		}
	}
	// Loki applies limit and direction globally, but entries are grouped by stream in the
	// response, so they must be re-sorted after merging.
	slices.SortStableFunc(entries, func(a, b lokiEntry) int {
		if direction == types.OrderDirectionAsc {
			return a.timestamp.Compare(b.timestamp)
		}

		return b.timestamp.Compare(a.timestamp)
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	logger.Debug("loki query_range response",
		zap.Int("entries", len(entries)), zap.Duration("duration", time.Since(began)))

	return entries, nil
}

// series returns all distinct values of the resource label among streams matching the
// given selector in the given time range (both bounds inclusive).
func (s *lokiStore) series(
	ctx context.Context,
	orgID uuid.UUID,
	selector string,
	start, end time.Time,
) ([]string, error) {
	params := url.Values{
		"match[]": []string{selector},
		"start":   []string{strconv.FormatInt(start.UnixNano(), 10)},
		"end":     []string{strconv.FormatInt(end.UnixNano()+1, 10)},
	}

	req, err := s.newRequest(ctx, orgID, http.MethodGet, "/loki/api/v1/series", params)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("loki series query failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("loki series query failed with status %v: %v", resp.StatusCode, readErrorBody(resp))
	}

	var body struct {
		Data []map[string]string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("could not decode loki series response: %w", err)
	}

	resourceSet := map[string]struct{}{}
	for _, labels := range body.Data {
		if resource, ok := labels[labelResource]; ok {
			resourceSet[resource] = struct{}{}
		}
	}

	resources := make([]string, 0, len(resourceSet))
	for resource := range resourceSet {
		resources = append(resources, resource)
	}

	return resources, nil
}

func (s *lokiStore) newRequest(
	ctx context.Context,
	orgID uuid.UUID,
	method, path string,
	params url.Values,
) (*http.Request, error) {
	requestURL := strings.TrimSuffix(s.config.URL, "/") + path
	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create loki request: %w", err)
	}

	req.Header.Set("X-Scope-OrgID", orgID.String())
	if s.config.BearerToken != nil {
		req.Header.Set("Authorization", "Bearer "+*s.config.BearerToken)
	} else if s.config.BasicAuthUsername != nil && s.config.BasicAuthPassword != nil {
		req.SetBasicAuth(*s.config.BasicAuthUsername, *s.config.BasicAuthPassword)
	}

	return req, nil
}

func readErrorBody(resp *http.Response) string {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return strings.TrimSpace(string(data))
}

func deploymentSelector(deploymentID uuid.UUID, resources []string) string {
	var b strings.Builder

	fmt.Fprintf(&b, `{%s=%q,%s=%q`, labelKind, kindDeployment, labelDeploymentID, deploymentID)

	if len(resources) == 1 {
		fmt.Fprintf(&b, `,%s=%q`, labelResource, resources[0])
	} else if len(resources) > 1 {
		quoted := make([]string, len(resources))
		for i, resource := range resources {
			quoted[i] = regexp.QuoteMeta(resource)
		}
		// Label regex matchers are fully anchored in LogQL.
		fmt.Fprintf(&b, `,%s=~%q`, labelResource, strings.Join(quoted, "|"))
	}

	b.WriteString(`}`)
	return b.String()
}

func deploymentRevisionsSelector(deploymentID uuid.UUID, revisionIDs []uuid.UUID) string {
	ids := make([]string, len(revisionIDs))
	for i, id := range revisionIDs {
		ids[i] = id.String()
	}

	return fmt.Sprintf(`{%s=%q,%s=%q,%s=~%q}`,
		labelKind, kindDeployment,
		labelDeploymentID, deploymentID,
		labelDeploymentRevisionID, strings.Join(ids, "|"))
}

func deploymentTargetSelector(deploymentTargetID uuid.UUID) string {
	return fmt.Sprintf(`{%s=%q,%s=%q}`,
		labelKind, kindDeploymentTarget,
		labelDeploymentTargetID, deploymentTargetID)
}

// filterExpr renders the user-supplied body filter as a LogQL regex line filter. The
// filter is validated as RE2 by the handler, but must still be quoted so it cannot break
// out of the string literal.
func filterExpr(filter string) string {
	if filter == "" {
		return ""
	}

	return ` |~ ` + strconv.Quote(filter)
}

func entryToDeploymentLogRecord(entry lokiEntry) (DeploymentLogRecord, error) {
	deploymentID, err := uuid.Parse(entry.labels[labelDeploymentID])
	if err != nil {
		return DeploymentLogRecord{}, fmt.Errorf("could not parse %v label: %w", labelDeploymentID, err)
	}

	revisionID, err := uuid.Parse(entry.labels[labelDeploymentRevisionID])
	if err != nil {
		return DeploymentLogRecord{}, fmt.Errorf("could not parse %v label: %w", labelDeploymentRevisionID, err)
	}

	record := DeploymentLogRecord{
		DeploymentID:         deploymentID,
		DeploymentRevisionID: revisionID,
		Resource:             entry.labels[labelResource],
		Timestamp:            entry.timestamp,
		Severity:             entry.labels[metadataSeverity],
		Body:                 entry.line,
	}

	record.ID = DeploymentLogRecordID(record)

	return record, nil
}

func entryToDeploymentTargetLogRecord(entry lokiEntry) (DeploymentTargetLogRecord, error) {
	deploymentTargetID, err := uuid.Parse(entry.labels[labelDeploymentTargetID])
	if err != nil {
		return DeploymentTargetLogRecord{}, fmt.Errorf("could not parse %v label: %w", labelDeploymentTargetID, err)
	}

	record := DeploymentTargetLogRecord{
		DeploymentTargetID: deploymentTargetID,
		Timestamp:          entry.timestamp,
		Severity:           entry.labels[metadataSeverity],
		Body:               entry.line,
	}

	record.ID = DeploymentTargetLogRecordID(record)

	return record, nil
}
