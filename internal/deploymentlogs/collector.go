package deploymentlogs

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type DeploymentIDer interface {
	GetDeploymentID() uuid.UUID
	GetDeploymentRevisionID() uuid.UUID
}

type Collector interface {
	For(DeploymentIDer) DeploymentCollector
	Flush(context.Context) error
}

type DeploymentCollector interface {
	AppendMessage(ctx context.Context, resource, severity, message string) error
}

type collector struct {
	mut             *sync.Mutex
	flushLimit      int
	bufferSizeLimit int
	exporter        Exporter
	log             *zap.Logger
	logRecords      []api.DeploymentLogRecord
}

const (
	defaultFlushLimit      = 500
	defaultBufferSizeLimit = 1000
)

func NewCollector(exporter Exporter, log *zap.Logger) Collector {
	return &collector{
		mut:             new(sync.Mutex),
		flushLimit:      defaultFlushLimit,
		bufferSizeLimit: defaultBufferSizeLimit,
		exporter:        exporter,
		log:             log,
		logRecords:      make([]api.DeploymentLogRecord, 0, defaultFlushLimit),
	}
}

// For implements Collector.
func (c *collector) For(d DeploymentIDer) DeploymentCollector {
	return &deploymentCollector{collector: c, DeploymentIDer: d}
}

func (c *collector) Flush(ctx context.Context) error {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.flushNoLock(ctx)
}

func (c *collector) flushNoLock(ctx context.Context) error {
	log := c.log.With(zap.Int("logRecords", len(c.logRecords)), zap.Strings("resourceNames", c.resourceNames()))
	if len(c.logRecords) == 0 {
		log.Debug("skipped flushing log records")
		return nil
	}

	t := time.Now()
	if err := c.exporter.ExportDeploymentLogs(ctx, c.logRecords); err != nil {
		return fmt.Errorf("export log records: %w", err)
	} else {
		log.Debug("flushed log records", zap.Duration("duration", time.Since(t)))
		c.logRecords = make([]api.DeploymentLogRecord, 0, c.flushLimit)
	}

	return nil
}

func (c *collector) appendRecord(ctx context.Context, record api.DeploymentLogRecord) error {
	c.mut.Lock()
	defer c.mut.Unlock()

	if len(c.logRecords) >= c.bufferSizeLimit {
		return fmt.Errorf("buffer size limit of %v has been reached", c.bufferSizeLimit)
	}

	c.logRecords = append(c.logRecords, record)
	if c.flushLimit > 0 && len(c.logRecords) >= c.flushLimit {
		if err := c.flushNoLock(ctx); err != nil {
			c.log.Warn("failed to flush log records", zap.Error(err), zap.Int("logRecords", len(c.logRecords)))
		}
	}

	return nil
}

func (c *collector) resourceNames() []string {
	namesMap := make(map[string]struct{})
	for _, r := range c.logRecords {
		if _, ok := namesMap[r.Resource]; !ok {
			namesMap[r.Resource] = struct{}{}
		}
	}

	return slices.Sorted(maps.Keys(namesMap))
}

type deploymentCollector struct {
	*collector
	DeploymentIDer
}

// AppendMessage implements DeploymentCollector.
func (d *deploymentCollector) AppendMessage(ctx context.Context, resource, severity, message string) error {
	record := NewRecord(d.GetDeploymentID(), d.GetDeploymentRevisionID(), resource, severity, message)
	if record.Body != "" {
		if err := d.appendRecord(ctx, record); err != nil {
			return fmt.Errorf("append message: %w", err)
		}
	}
	return nil
}
