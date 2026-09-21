package deploymenttargetlogs

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/distr-sh/distr/api"
)

const (
	defaultBufferSize    = 128
	defaultMaxSize       = 1024
	defaultFlushInterval = 30 * time.Second
)

type BufferedCollector struct {
	Size          int
	MaxSize       int
	FlushInterval time.Duration
	Delegate      Exporter

	buf         []api.DeploymentTargetLogRecord
	mu          sync.Mutex
	initialized bool
	syncing     bool
	stop        chan struct{}
	done        chan struct{}
}

// ExportDeploymentTargetLogs implements [Exporter].
func (bc *BufferedCollector) ExportDeploymentTargetLogs(records ...api.DeploymentTargetLogRecord) error {
	bc.mu.Lock()
	if !bc.initialized {
		bc.init()
	}
	atMaxSize := bc.isMaxBufferSize()
	bc.mu.Unlock()

	if atMaxSize {
		if err := bc.Sync(); err != nil {
			// Max buffer size is reached and sync failed --> write error (the records will be lost!)
			return err
		}
	}

	bc.mu.Lock()
	if bc.isMaxBufferSize() {
		// The sync above did not drain the buffer because another one is still in flight, and the
		// records buffered in the meantime filled it --> write error (the records will be lost!)
		bc.mu.Unlock()
		return errors.New("log buffer is full")
	}
	bc.buf = append(bc.buf, records...)
	syncRequired := bc.isSyncRequired()
	bc.mu.Unlock()

	if syncRequired {
		if err := bc.Sync(); err != nil {
			// Do not return an error, because a failure to sync at this point does not indicate a write error.
			// Print an error to stderr, we can not use the zap logger here (zap does this too internally).
			fmt.Fprintf(os.Stderr, "%v sync error: %v\n", time.Now().Format(time.RFC3339), err)
		}
	}

	return nil
}

// Sync implements [Syncer].
func (bc *BufferedCollector) Sync() error {
	if bc.Delegate == nil {
		return errors.New("bufferedCollector has no Delegate")
	}

	bc.mu.Lock()
	if bc.syncing || len(bc.buf) == 0 {
		// A record appended while the delegate is exporting belongs to the next batch. Exporting it
		// from here would recurse through the delegate, which logs and therefore ends up back here.
		bc.mu.Unlock()
		return nil
	}
	batch := bc.buf
	bc.syncing = true
	bc.resetBuffer()
	bc.mu.Unlock()

	// The delegate must not be called with the lock held: it logs, the agent's logger writes into
	// this collector and the lock is not reentrant, so that would deadlock the logging goroutine.
	err := bc.Delegate.ExportDeploymentTargetLogs(batch...)

	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.syncing = false

	if err != nil {
		if errors.Is(err, ErrRecordsRejected) {
			// The server permanently rejected these records. Drop them so newer logs
			// keep flowing; retrying would fail forever and wedge the buffer.
			// We cannot use the zap logger here (this collector is a zap sink).
			fmt.Fprintf(os.Stderr, "%v dropping log records rejected by the server: %v\n",
				time.Now().Format(time.RFC3339), err)
			return nil
		}
		bc.buf = append(batch, bc.buf...)
		return err
	}

	return nil
}

func (bc *BufferedCollector) Stop() error {
	close(bc.stop)
	<-bc.done
	return bc.Sync()
}

func (bc *BufferedCollector) init() {
	bc.initialized = true

	bc.resetBuffer()

	syncInterval := bc.FlushInterval
	if syncInterval == 0 {
		syncInterval = defaultFlushInterval
	}

	bc.stop = make(chan struct{})
	bc.done = make(chan struct{})

	tick := time.Tick(syncInterval)
	go func() {
		defer close(bc.done)
		for {
			select {
			case <-tick:
				_ = bc.Sync()
			case <-bc.stop:
				return
			}
		}
	}()
}

func (bc *BufferedCollector) sizeOrDefault() int {
	if bc.Size != 0 {
		return bc.Size
	}
	return defaultBufferSize
}

func (bc *BufferedCollector) maxSizeOrDefault() int {
	if bc.MaxSize != 0 {
		return bc.MaxSize
	}
	return defaultMaxSize
}

func (bc *BufferedCollector) resetBuffer() {
	bc.buf = make([]api.DeploymentTargetLogRecord, 0, bc.sizeOrDefault())
}

func (bc *BufferedCollector) isSyncRequired() bool {
	return len(bc.buf) >= bc.sizeOrDefault()
}

func (bc *BufferedCollector) isMaxBufferSize() bool {
	return len(bc.buf) >= bc.maxSizeOrDefault()
}

var (
	_ Exporter = (*BufferedCollector)(nil)
	_ Syncer   = (*BufferedCollector)(nil)
)
