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
	stop        chan struct{}
	done        chan struct{}
}

// ExportDeploymentTargetLogs implements [Exporter].
func (bc *BufferedCollector) ExportDeploymentTargetLogs(records ...api.DeploymentTargetLogRecord) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if !bc.initialized {
		bc.init()
	}

	for _, record := range records {
		if err := bc.appendBuffer(record); err != nil {
			return err
		}
	}

	return nil
}

// Sync implements [Syncer].
func (bc *BufferedCollector) Sync() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.syncNoLock()
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

func (bc *BufferedCollector) appendBuffer(record api.DeploymentTargetLogRecord) error {
	if bc.isMaxBufferSize() {
		if err := bc.syncNoLock(); err != nil {
			// Max buffer size is reached and sync failed --> write error (the record will be lost!)
			return err
		}
	}

	bc.buf = append(bc.buf, record)

	if bc.isSyncRequired() {
		if err := bc.syncNoLock(); err != nil {
			// Do not return an error, because a failure to sync at this point does not indicate a write error.
			// Print an error to stderr, we can not use the zap logger here (zap does this too internally).
			fmt.Fprintf(os.Stderr, "%v sync error: %v\n", time.Now(), err)
		}
	}
	return nil
}

func (bc *BufferedCollector) syncNoLock() error {
	if bc.Delegate == nil {
		return errors.New("bufferedCollector has no Delegate")
	}
	if len(bc.buf) > 0 {
		if err := bc.Delegate.ExportDeploymentTargetLogs(bc.buf...); err != nil {
			return err
		}
		bc.resetBuffer()
	}
	return nil
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
