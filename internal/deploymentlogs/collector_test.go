package deploymentlogs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
)

type fakeExporter struct {
	calls     [][]api.DeploymentLogRecord
	err       error
	errOnce   bool
	callCount int
}

func (f *fakeExporter) ExportDeploymentLogs(_ context.Context, records []api.DeploymentLogRecord) error {
	f.callCount++
	f.calls = append(f.calls, append([]api.DeploymentLogRecord(nil), records...))
	if f.err != nil && (!f.errOnce || f.callCount == 1) {
		return f.err
	}
	return nil
}

type fakeDeployment struct {
	id, revision uuid.UUID
}

func (f fakeDeployment) GetDeploymentID() uuid.UUID         { return f.id }
func (f fakeDeployment) GetDeploymentRevisionID() uuid.UUID { return f.revision }

func newTestCollector(t *testing.T, exporter Exporter, flushLimit, bufferSizeLimit int) *collector {
	t.Helper()
	return &collector{
		mut:             new(sync.Mutex),
		flushLimit:      flushLimit,
		bufferSizeLimit: bufferSizeLimit,
		exporter:        exporter,
		log:             zaptest.NewLogger(t),
		logRecords:      make([]api.DeploymentLogRecord, 0, flushLimit),
	}
}

func TestCollectorDropsRejectedRecords(t *testing.T) {
	g := NewWithT(t)
	exporter := &fakeExporter{err: fmt.Errorf("%w: bad line", ErrRecordsRejected)}
	c := newTestCollector(t, exporter, 2, 10)
	dc := c.For(fakeDeployment{id: uuid.New(), revision: uuid.New()})

	// Appending up to flushLimit triggers a flush that the server rejects. The batch
	// must be dropped (buffer cleared) and AppendMessage must not surface an error, so
	// collection can continue instead of poison-blocking.
	g.Expect(dc.AppendMessage(t.Context(), "resource", "info", "a")).To(Succeed())
	g.Expect(dc.AppendMessage(t.Context(), "resource", "info", "b")).To(Succeed())

	g.Expect(exporter.callCount).To(Equal(1))
	g.Expect(c.logRecords).To(BeEmpty())

	// A subsequent flush has nothing buffered and must be a no-op.
	g.Expect(c.Flush(t.Context())).To(Succeed())
	g.Expect(exporter.callCount).To(Equal(1))
}

func TestCollectorNeverPoisonBlocksOnRejection(t *testing.T) {
	g := NewWithT(t)
	exporter := &fakeExporter{err: ErrRecordsRejected}
	c := newTestCollector(t, exporter, 2, 4)
	dc := c.For(fakeDeployment{id: uuid.New(), revision: uuid.New()})

	// Many rejected records in a row must never fill the buffer to bufferSizeLimit (which
	// would make AppendMessage error and cancel collection) nor surface an error.
	for range 20 {
		g.Expect(dc.AppendMessage(t.Context(), "resource", "info", "x")).To(Succeed())
		g.Expect(len(c.logRecords)).To(BeNumerically("<", c.bufferSizeLimit))
	}
}

func TestCollectorRetainsRecordsOnTransientError(t *testing.T) {
	g := NewWithT(t)
	exporter := &fakeExporter{err: errors.New("connection refused"), errOnce: true}
	c := newTestCollector(t, exporter, 2, 10)
	dc := c.For(fakeDeployment{id: uuid.New(), revision: uuid.New()})

	// A transient flush failure must retain the records so they are retried, not dropped.
	g.Expect(dc.AppendMessage(t.Context(), "resource", "info", "a")).To(Succeed())
	g.Expect(dc.AppendMessage(t.Context(), "resource", "info", "b")).To(Succeed())
	g.Expect(exporter.callCount).To(Equal(1))
	g.Expect(c.logRecords).To(HaveLen(2))

	// The next flush succeeds and resends the retained records, then clears the buffer.
	g.Expect(c.Flush(t.Context())).To(Succeed())
	g.Expect(exporter.callCount).To(Equal(2))
	g.Expect(exporter.calls[1]).To(HaveLen(2))
	g.Expect(c.logRecords).To(BeEmpty())
}

func TestCollectorFlushReturnsTransientError(t *testing.T) {
	g := NewWithT(t)
	exporter := &fakeExporter{err: errors.New("connection refused")}
	c := newTestCollector(t, exporter, 1000, 2000)
	dc := c.For(fakeDeployment{id: uuid.New(), revision: uuid.New()})

	g.Expect(dc.AppendMessage(t.Context(), "resource", "info", "a")).To(Succeed())

	// A transient error surfaces from an explicit Flush (so callers can avoid advancing
	// their watermark), and the records stay buffered for a retry.
	g.Expect(c.Flush(t.Context())).NotTo(Succeed())
	g.Expect(c.logRecords).To(HaveLen(1))
}
