package deploymenttargetlogs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/distr-sh/distr/api"
	. "github.com/onsi/gomega"
)

type fakeDelegate struct {
	calls     [][]api.DeploymentTargetLogRecord
	err       error
	errOnce   bool
	callCount int
}

func (f *fakeDelegate) ExportDeploymentTargetLogs(records ...api.DeploymentTargetLogRecord) error {
	f.callCount++
	f.calls = append(f.calls, append([]api.DeploymentTargetLogRecord(nil), records...))
	if f.err != nil && (!f.errOnce || f.callCount == 1) {
		return f.err
	}
	return nil
}

// newTestBufferedCollector returns a collector that is already initialized without
// starting the background flush goroutine, so tests can drive syncs deterministically.
func newTestBufferedCollector(delegate Exporter, size, maxSize int) *BufferedCollector {
	bc := &BufferedCollector{Delegate: delegate, Size: size, MaxSize: maxSize}
	bc.initialized = true
	bc.resetBuffer()
	return bc
}

func TestBufferedCollectorDropsRejectedRecords(t *testing.T) {
	g := NewWithT(t)
	delegate := &fakeDelegate{err: fmt.Errorf("%w: bad line", ErrRecordsRejected)}
	bc := newTestBufferedCollector(delegate, 100, 100)

	g.Expect(bc.ExportDeploymentTargetLogs(api.DeploymentTargetLogRecord{Body: "a"})).To(Succeed())

	// The server rejects the batch; Sync must drop the records (clear the buffer) and
	// not surface an error, so newer logs keep flowing instead of wedging the buffer.
	g.Expect(bc.Sync()).To(Succeed())
	g.Expect(delegate.callCount).To(Equal(1))
	g.Expect(bc.buf).To(BeEmpty())

	// A subsequent Sync has nothing buffered and must be a no-op.
	g.Expect(bc.Sync()).To(Succeed())
	g.Expect(delegate.callCount).To(Equal(1))
}

func TestBufferedCollectorNeverWedgesOnRejection(t *testing.T) {
	g := NewWithT(t)
	delegate := &fakeDelegate{err: ErrRecordsRejected}
	bc := newTestBufferedCollector(delegate, 4, 4)

	// Many rejected records in a row must never fill the buffer to MaxSize (which would
	// make ExportDeploymentTargetLogs error and drop newer records) nor surface an error.
	for range 20 {
		g.Expect(bc.ExportDeploymentTargetLogs(api.DeploymentTargetLogRecord{Body: "x"})).To(Succeed())
		g.Expect(len(bc.buf)).To(BeNumerically("<", bc.maxSizeOrDefault()))
	}
}

func TestBufferedCollectorRetainsRecordsOnTransientError(t *testing.T) {
	g := NewWithT(t)
	delegate := &fakeDelegate{err: errors.New("connection refused"), errOnce: true}
	bc := newTestBufferedCollector(delegate, 100, 100)

	g.Expect(bc.ExportDeploymentTargetLogs(api.DeploymentTargetLogRecord{Body: "a"})).To(Succeed())

	// A transient failure must retain the records and surface the error so they are retried.
	g.Expect(bc.Sync()).NotTo(Succeed())
	g.Expect(bc.buf).To(HaveLen(1))

	// The next Sync succeeds, resends the retained record, then clears the buffer.
	g.Expect(bc.Sync()).To(Succeed())
	g.Expect(delegate.callCount).To(Equal(2))
	g.Expect(delegate.calls[1]).To(HaveLen(1))
	g.Expect(bc.buf).To(BeEmpty())
}
