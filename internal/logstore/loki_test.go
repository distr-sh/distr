package logstore

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

var (
	testOrgID              = uuid.MustParse("998fea34-1697-4ffa-be96-2d19f8b73a1c")
	testDeploymentID       = uuid.MustParse("98be36e4-aa8a-4596-a5e8-8da0e0974105")
	testRevisionID         = uuid.MustParse("addb2eac-c1e5-4580-a36e-42c011327dd5")
	testDeploymentTargetID = uuid.MustParse("bd3ff37e-9dc2-4de6-9668-3f0e4c112233")
)

func newTestStore(t *testing.T, handler http.Handler) *lokiStore {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	store, err := NewLokiStore(LokiConfig{URL: server.URL})
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
	return store.(*lokiStore)
}

type pushRequest struct {
	Streams []struct {
		Stream map[string]string `json:"stream"`
		Values [][]any           `json:"values"`
	} `json:"streams"`
}

func TestSaveDeploymentLogRecords(t *testing.T) {
	g := NewWithT(t)

	var captured pushRequest
	var tenant string
	store := newTestStore(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.URL.Path).To(Equal("/loki/api/v1/push"))
		tenant = r.Header.Get("X-Scope-OrgID")
		g.Expect(json.NewDecoder(r.Body).Decode(&captured)).To(Succeed())
		w.WriteHeader(http.StatusNoContent)
	}))

	timestamp := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	err := store.SaveDeploymentLogRecords(t.Context(), testOrgID, []api.DeploymentLogRecord{
		{
			DeploymentID:         testDeploymentID,
			DeploymentRevisionID: testRevisionID,
			Resource:             "resource-a",
			Timestamp:            timestamp,
			Severity:             "info",
			Body:                 "first line",
		},
		{
			DeploymentID:         testDeploymentID,
			DeploymentRevisionID: testRevisionID,
			Resource:             "resource-a",
			Timestamp:            timestamp.Add(time.Second),
			Severity:             "error",
			Body:                 "second line",
		},
		{
			DeploymentID:         testDeploymentID,
			DeploymentRevisionID: testRevisionID,
			Resource:             "resource-b",
			Timestamp:            timestamp,
			Severity:             "info",
			Body:                 "other resource",
		},
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(tenant).To(Equal(testOrgID.String()))

	// Records with identical labels must be grouped into a single stream.
	g.Expect(captured.Streams).To(HaveLen(2))
	byResource := map[string][][]any{}
	for _, stream := range captured.Streams {
		g.Expect(stream.Stream).To(HaveKeyWithValue("distr_kind", "deployment"))
		g.Expect(stream.Stream).To(HaveKeyWithValue("deployment_id", testDeploymentID.String()))
		g.Expect(stream.Stream).To(HaveKeyWithValue("deployment_revision_id", testRevisionID.String()))
		byResource[stream.Stream["resource"]] = stream.Values
	}
	g.Expect(byResource["resource-a"]).To(HaveLen(2))
	g.Expect(byResource["resource-b"]).To(HaveLen(1))

	// Severity is sent as structured metadata (third value element).
	first := byResource["resource-a"][0]
	g.Expect(first).To(HaveLen(3))
	g.Expect(first[0]).To(Equal(strconv.FormatInt(timestamp.UnixNano(), 10)))
	g.Expect(first[1]).To(Equal("first line"))
	g.Expect(first[2]).To(Equal(map[string]any{"severity": "info"}))
}

func TestSaveDeploymentLogRecordsBadRequest(t *testing.T) {
	g := NewWithT(t)
	store := newTestStore(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "entry too far behind", http.StatusBadRequest)
	}))
	err := store.SaveDeploymentLogRecords(t.Context(), testOrgID, []api.DeploymentLogRecord{{Body: "x"}})
	g.Expect(err).To(MatchError(ContainSubstring("bad request")))
	g.Expect(err).To(MatchError(ContainSubstring("entry too far behind")))
}

// mockEntry is a log entry served by the fake query_range handler.
type mockEntry struct {
	labels map[string]string
	ts     time.Time
	line   string
}

// queryRangeHandler emulates Loki's query_range endpoint for a fixed set of entries:
// start is inclusive, end is exclusive, entries are sorted according to direction,
// truncated to limit, and grouped by stream labels in the response.
func queryRangeHandler(t *testing.T, entries []mockEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/loki/api/v1/query_range" {
			t.Errorf("unexpected path: %v", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		query := r.URL.Query()
		start, _ := strconv.ParseInt(query.Get("start"), 10, 64)
		end, _ := strconv.ParseInt(query.Get("end"), 10, 64)
		limit, _ := strconv.Atoi(query.Get("limit"))
		direction := query.Get("direction")

		var selected []mockEntry
		for _, entry := range entries {
			if ns := entry.ts.UnixNano(); ns >= start && ns < end {
				selected = append(selected, entry)
			}
		}
		slices.SortStableFunc(selected, func(a, b mockEntry) int {
			if direction == "forward" {
				return cmp.Compare(a.ts.UnixNano(), b.ts.UnixNano())
			}
			return cmp.Compare(b.ts.UnixNano(), a.ts.UnixNano())
		})
		if len(selected) > limit {
			selected = selected[:limit]
		}

		type stream struct {
			Stream map[string]string `json:"stream"`
			Values [][]string        `json:"values"`
		}
		streams := map[string]*stream{}
		var order []string
		for _, entry := range selected {
			key := streamKey(entry.labels)
			s, ok := streams[key]
			if !ok {
				s = &stream{Stream: entry.labels}
				streams[key] = s
				order = append(order, key)
			}
			s.Values = append(s.Values, []string{strconv.FormatInt(entry.ts.UnixNano(), 10), entry.line})
		}
		result := make([]*stream, 0, len(order))
		for _, key := range order {
			result = append(result, streams[key])
		}

		response := map[string]any{
			"status": "success",
			"data":   map[string]any{"resultType": "streams", "result": result},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}

func deploymentTargetLabels(severity string) map[string]string {
	return map[string]string{
		"distr_kind":           "deployment_target",
		"deployment_target_id": testDeploymentTargetID.String(),
		"severity":             severity,
	}
}

func TestQueryDeploymentTargetLogRecords(t *testing.T) {
	g := NewWithT(t)
	base := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	entries := []mockEntry{
		{labels: deploymentTargetLabels("info"), ts: base, line: "oldest"},
		{labels: deploymentTargetLabels("error"), ts: base.Add(time.Second), line: "middle"},
		{labels: deploymentTargetLabels("info"), ts: base.Add(2 * time.Second), line: "newest"},
	}
	store := newTestStore(t, queryRangeHandler(t, entries))

	records, err := util.SeqCollect(store.QueryDeploymentTargetLogRecords(t.Context(), testOrgID, DeploymentTargetLogQuery{
		DeploymentTargetID: testDeploymentTargetID,
		Start:              base,
		End:                base.Add(2 * time.Second),
		Limit:              10,
		Direction:          types.OrderDirectionDesc,
	}))
	g.Expect(err).NotTo(HaveOccurred())

	// Streams (split by severity) must be merged and globally sorted newest first,
	// with both range bounds inclusive.
	g.Expect(records).To(HaveLen(3))
	g.Expect(records[0].Body).To(Equal("newest"))
	g.Expect(records[1].Body).To(Equal("middle"))
	g.Expect(records[1].Severity).To(Equal("error"))
	g.Expect(records[2].Body).To(Equal("oldest"))
	g.Expect(records[0].DeploymentTargetID).To(Equal(testDeploymentTargetID))
	g.Expect(records[0].ID).To(Equal(DeploymentTargetLogRecordID(records[0])))
	g.Expect(records[0].ID).NotTo(Equal(records[1].ID))
}

func TestQueryDeploymentTargetLogRecordsAscending(t *testing.T) {
	g := NewWithT(t)
	base := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	entries := []mockEntry{
		{labels: deploymentTargetLabels("info"), ts: base.Add(time.Second), line: "second"},
		{labels: deploymentTargetLabels("info"), ts: base, line: "first"},
	}
	store := newTestStore(t, queryRangeHandler(t, entries))

	records, err := util.SeqCollect(store.QueryDeploymentTargetLogRecords(t.Context(), testOrgID, DeploymentTargetLogQuery{
		DeploymentTargetID: testDeploymentTargetID,
		Start:              base,
		End:                base.Add(time.Minute),
		Limit:              10,
		Direction:          types.OrderDirectionAsc,
	}))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(records).To(HaveLen(2))
	g.Expect(records[0].Body).To(Equal("first"))
	g.Expect(records[1].Body).To(Equal("second"))
}

func TestQueryDeploymentLogRecordsSelector(t *testing.T) {
	g := NewWithT(t)
	var capturedQuery string
	store := newTestStore(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("query")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"status":"success","data":{"resultType":"streams","result":[]}}`)
	}))

	_, err := util.SeqCollect(store.QueryDeploymentLogRecords(t.Context(), testOrgID, DeploymentLogQuery{
		DeploymentID: testDeploymentID,
		Resources:    []string{"resource-a", "resource+b"},
		Start:        time.Now().Add(-time.Hour),
		End:          time.Now(),
		Filter:       "some.*filter",
		Limit:        10,
		Direction:    types.OrderDirectionDesc,
	}))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(capturedQuery).To(Equal(
		`{distr_kind="deployment",deployment_id="` + testDeploymentID.String() +
			`",resource=~"resource-a|resource\\+b"} |~ "some.*filter"`,
	))
}

// pagingTestEntries builds 10 entries across two streams where two pairs share the
// exact same nanosecond across streams. With a page size of 3, the paging boundary
// lands on shared timestamps, exercising the fingerprint-based de-duplication.
func pagingTestEntries(base time.Time) []mockEntry {
	deploymentLabels := func(resource string) map[string]string {
		return map[string]string{
			"distr_kind":             "deployment",
			"deployment_id":          testDeploymentID.String(),
			"deployment_revision_id": testRevisionID.String(),
			"resource":               resource,
			"severity":               "info",
		}
	}
	labelsA := deploymentLabels("resource-a")
	labelsB := deploymentLabels("resource-b")
	return []mockEntry{
		{labels: labelsA, ts: base.Add(0), line: "a-0"},
		{labels: labelsB, ts: base.Add(0), line: "b-0"},
		{labels: labelsA, ts: base.Add(1 * time.Second), line: "a-1"},
		{labels: labelsA, ts: base.Add(2 * time.Second), line: "a-2"},
		{labels: labelsB, ts: base.Add(2 * time.Second), line: "b-2"},
		{labels: labelsA, ts: base.Add(3 * time.Second), line: "a-3"},
		{labels: labelsB, ts: base.Add(4 * time.Second), line: "b-4"},
		{labels: labelsA, ts: base.Add(5 * time.Second), line: "a-5"},
		{labels: labelsB, ts: base.Add(6 * time.Second), line: "b-6"},
		{labels: labelsA, ts: base.Add(7 * time.Second), line: "a-7"},
	}
}

func TestQueryDeploymentLogRecordsPaging(t *testing.T) {
	g := NewWithT(t)
	// Reads with a zero End query up to time.Now(), so the entries must lie in the past.
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	entries := pagingTestEntries(base)

	requestCount := 0
	handler := queryRangeHandler(t, entries)
	store := newTestStore(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		handler(w, r)
	}))
	store.maxEntriesPerQuery = 3

	var bodies []string
	seq := store.QueryDeploymentLogRecords(t.Context(), testOrgID, DeploymentLogQuery{
		DeploymentID: testDeploymentID,
		Resources:    []string{"resource-a", "resource-b"},
		Start:        base,
		Limit:        100,
		Direction:    types.OrderDirectionDesc,
	})
	for record, err := range seq {
		g.Expect(err).NotTo(HaveOccurred())
		bodies = append(bodies, record.Body)
	}

	// All entries must be returned exactly once, newest first, despite page
	// boundaries landing on timestamps shared across streams.
	g.Expect(bodies).To(HaveLen(len(entries)))
	g.Expect(bodies[0]).To(Equal("a-7"))
	g.Expect(bodies[len(bodies)-1:]).To(ConsistOf(BeElementOf("a-0", "b-0")))
	seen := map[string]struct{}{}
	for _, body := range bodies {
		g.Expect(seen).NotTo(HaveKey(body), "duplicate entry %s", body)
		seen[body] = struct{}{}
	}
	g.Expect(requestCount).To(BeNumerically(">", 1))
}

func TestQueryDeploymentLogRecordsPagingAscending(t *testing.T) {
	g := NewWithT(t)
	// Same fixture as the descending paging test: timestamps shared across streams land
	// on page boundaries, exercising the forward-paging fingerprint de-duplication.
	base := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	entries := pagingTestEntries(base)

	requestCount := 0
	handler := queryRangeHandler(t, entries)
	store := newTestStore(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		handler(w, r)
	}))
	store.maxEntriesPerQuery = 3

	var bodies []string
	seq := store.QueryDeploymentLogRecords(t.Context(), testOrgID, DeploymentLogQuery{
		DeploymentID: testDeploymentID,
		Resources:    []string{"resource-a", "resource-b"},
		Start:        base,
		End:          base.Add(time.Minute),
		Limit:        100,
		Direction:    types.OrderDirectionAsc,
	})
	for record, err := range seq {
		g.Expect(err).NotTo(HaveOccurred())
		bodies = append(bodies, record.Body)
	}

	// All entries must be returned exactly once, oldest first.
	g.Expect(bodies).To(HaveLen(len(entries)))
	g.Expect(bodies[:1]).To(ConsistOf(BeElementOf("a-0", "b-0")))
	g.Expect(bodies[len(bodies)-1]).To(Equal("a-7"))
	seen := map[string]struct{}{}
	for _, body := range bodies {
		g.Expect(seen).NotTo(HaveKey(body), "duplicate entry %s", body)
		seen[body] = struct{}{}
	}
	g.Expect(requestCount).To(BeNumerically(">", 1))
}

func TestQueryDeploymentLogRecordsLimit(t *testing.T) {
	g := NewWithT(t)
	// Streaming reads query up to time.Now(), so the entries must lie in the past.
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	labels := map[string]string{
		"distr_kind":             "deployment",
		"deployment_id":          testDeploymentID.String(),
		"deployment_revision_id": testRevisionID.String(),
		"resource":               "resource-a",
		"severity":               "info",
	}
	entries := make([]mockEntry, 0, 10)
	for i := range 10 {
		entries = append(entries, mockEntry{
			labels: labels,
			ts:     base.Add(time.Duration(i) * time.Second),
			line:   fmt.Sprintf("line-%d", i),
		})
	}
	store := newTestStore(t, queryRangeHandler(t, entries))
	store.maxEntriesPerQuery = 4

	var bodies []string
	seq := store.QueryDeploymentLogRecords(t.Context(), testOrgID, DeploymentLogQuery{
		DeploymentID: testDeploymentID,
		Resources:    []string{"resource-a"},
		Start:        base,
		Limit:        7,
		Direction:    types.OrderDirectionDesc,
	})
	for record, err := range seq {
		g.Expect(err).NotTo(HaveOccurred())
		bodies = append(bodies, record.Body)
	}
	g.Expect(bodies).To(Equal([]string{
		"line-9", "line-8", "line-7", "line-6", "line-5", "line-4", "line-3",
	}))
}

func TestGetDeploymentLogRecordResources(t *testing.T) {
	g := NewWithT(t)
	store := newTestStore(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.URL.Path).To(Equal("/loki/api/v1/series"))
		match := r.URL.Query().Get("match[]")
		var data []map[string]string
		if strings.Contains(match, "deployment_revision_id") {
			data = []map[string]string{
				{"distr_kind": "deployment", "resource": "active-b"},
				{"distr_kind": "deployment", "resource": "active-a"},
			}
		} else {
			data = []map[string]string{
				{"distr_kind": "deployment", "resource": "active-a"},
				{"distr_kind": "deployment", "resource": "active-b"},
				{"distr_kind": "deployment", "resource": "archived-z"},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data})
	}))

	active, archived, err := store.GetDeploymentLogRecordResources(t.Context(), testOrgID, DeploymentLogResourcesQuery{
		DeploymentID:      testDeploymentID,
		LatestRevisionIDs: []uuid.UUID{testRevisionID},
		Start:             time.Now().Add(-time.Hour),
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(active).To(Equal([]string{"active-a", "active-b"}))
	g.Expect(archived).To(Equal([]string{"archived-z"}))
}
