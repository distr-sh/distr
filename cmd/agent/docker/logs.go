package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"time"

	"github.com/distr-sh/distr/internal/deploymentlogs"
	"github.com/distr-sh/distr/internal/types"
	"github.com/docker/cli/cli/compose/convert"
	composeapi "github.com/docker/compose/v5/pkg/api"
	"github.com/moby/moby/api/pkg/stdcopy"
	mobyClient "github.com/moby/moby/client"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type logsWatcher struct{}

func NewLogsWatcher() *logsWatcher {
	return &logsWatcher{}
}

func (lw *logsWatcher) Watch(ctx context.Context, d time.Duration) {
	logger.Debug("logs watcher is starting to watch",
		zap.Duration("interval", d))
	tick := time.Tick(d)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			lw.collect(ctx)
		}
	}
}

func (lw *logsWatcher) collect(ctx context.Context) {
	logger.Debug("getting logs")

	deployments, err := GetExistingDeployments()
	if err != nil {
		logger.Warn("watch logs could not get deployments", zap.Error(err))
		return
	}

	collector := deploymentlogs.NewCollector(client, logger)

	for _, d := range deployments {
		if !d.LogsEnabled {
			continue
		}

		deploymentCollector := collector.For(d)
		now := time.Now()
		var toplevelErr error

		since, err := GetLastLogsTimestamp(d)
		if err != nil {
			logger.Warn("could not get last logs timestamp", zap.Error(err))
		}

		if d.LogsAfter != nil && since == nil || since.Before(*d.LogsAfter) {
			since = d.LogsAfter
		}

		switch d.DockerType {
		case types.DockerTypeCompose:
			logOptions := composeapi.LogOptions{Timestamps: true}
			if since != nil {
				logOptions.Since = since.Format(time.RFC3339Nano)
			}

			// Allow the collector to cancel the context used for the API request.
			// This allows propagating append errors downstream.
			ctx, cancel := context.WithCancelCause(ctx)
			defer cancel(nil)
			collector := composeCollector{ctx, cancel, deploymentCollector}
			toplevelErr = composeService.Logs(ctx, d.ProjectName, &collector, logOptions)
			if toplevelErr != nil {
				logger.Warn("could not get compose logs", zap.Error(toplevelErr))
			}
		case types.DockerTypeSwarm:
			// Since there is no "docker stack logs" we have to take a small detour:
			// Getting the list of swarm services for the stack and then getting the logs for each service.
			// Because we are interacting with the API directly, we also have to decode the raw stream into its
			// stdout and stderr components.
			apiClient := dockerCli.Client()
			services, err := apiClient.ServiceList(
				ctx,
				mobyClient.ServiceListOptions{
					Filters: mobyClient.Filters{}.Add("label", convert.LabelNamespace+"="+d.ProjectName),
				},
			)
			if err != nil {
				logger.Warn("could not get services for docker stack", zap.Error(err))
				toplevelErr = err
			} else {
				for _, svc := range services.Items {
					// fake closure to close the ReadCloser returned by ServiceLogs after each iteration
					err := func() error {
						logOptions := mobyClient.ServiceLogsOptions{Timestamps: true, ShowStdout: true, ShowStderr: true}
						if since != nil {
							logOptions.Since = since.Format(time.RFC3339Nano)
						}
						rc, err := apiClient.ServiceLogs(ctx, svc.ID, logOptions)
						if err != nil {
							return err
						}
						defer rc.Close()
						return decodeServiceLogs(ctx, svc.Spec.Name, rc, deploymentCollector)
					}()
					if err != nil {
						logger.Warn("could not get service logs", zap.Error(err))
						toplevelErr = err
						break
					}
				}
			}
		}

		if toplevelErr == nil {
			if err := UpdateLastLogsTimestamp(d, now); err != nil {
				logger.Warn("could not update last logs timestamp for deployment", zap.Error(err))
			}
		}
	}

	if err := collector.Flush(ctx); err != nil {
		logger.Warn("error exporting logs", zap.Error(err))
	}
}

type composeCollector struct {
	ctx    context.Context
	cancel func(error)
	deploymentlogs.DeploymentCollector
}

// Err implements api.LogConsumer.
func (cc *composeCollector) Err(containerName, message string) {
	cc.LogWithSeverity(containerName, "Err", message)
}

// Log implements api.LogConsumer.
func (cc *composeCollector) Log(containerName, message string) {
	cc.LogWithSeverity(containerName, "Log", message)
}

func (cc *composeCollector) LogWithSeverity(containerName, severity, message string) {
	if err := cc.AppendMessage(cc.ctx, containerName, severity, message); err != nil {
		logger.Warn("failed to append log message", zap.Error(err))
		if cc.cancel != nil {
			cc.cancel(err)
		}
	}
}

// Register implements api.LogConsumer.
//
// Noop for now
func (*composeCollector) Register(containerName string) {}

// Status implements api.LogConsumer.
//
// Noop for now
func (*composeCollector) Status(containerName string, message string) {}

func decodeServiceLogs(
	ctx context.Context,
	resource string,
	r io.Reader,
	consumer deploymentlogs.DeploymentCollector,
) error {
	wg, ctx := errgroup.WithContext(ctx)

	scanAndCollect := func(pr *io.PipeReader, severity string) func() error {
		return func() error {
			scanner := bufio.NewScanner(pr)
			for scanner.Scan() {
				if err := consumer.AppendMessage(ctx, resource, severity, scanner.Text()); err != nil {
					pr.CloseWithError(err)
					return err
				}
			}
			if err := scanner.Err(); err != nil {
				pr.CloseWithError(err)
				return err
			}
			return nil
		}
	}

	wg.SetLimit(3)

	outReader, outWriter := io.Pipe()
	errReader, errWriter := io.Pipe()
	wg.Go(func() error {
		// The docker API provides a multiplexed stream for logs which must be demuxed. StdCopy does that.
		_, err := stdcopy.StdCopy(outWriter, errWriter, r)
		outWriter.CloseWithError(err)
		errWriter.CloseWithError(err)
		return err
	})

	wg.Go(scanAndCollect(outReader, "stdout"))
	wg.Go(scanAndCollect(errReader, "stderr"))

	return wg.Wait()
}

func UpdateLastLogsTimestamp(deployment AgentDeployment, timestamp time.Time) error {
	if err := os.MkdirAll(path.Dir(lastLogsTimestampFileName(deployment)), 0o700); err != nil {
		return err
	}

	file, err := os.Create(lastLogsTimestampFileName(deployment))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(timestamp.Format(time.RFC3339Nano))
	return err
}

func GetLastLogsTimestamp(deployment AgentDeployment) (*time.Time, error) {
	file, err := os.Open(lastLogsTimestampFileName(deployment))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer file.Close()

	if data, err := io.ReadAll(file); err != nil {
		return nil, err
	} else if ts, err := time.Parse(time.RFC3339Nano, string(data)); err != nil {
		return nil, err
	} else {
		return &ts, nil
	}
}

func CleanupLogsTimestamps(deployment AgentDeployment) {
	if err := os.Remove(lastLogsTimestampFileName(deployment)); err != nil {
		logger.Warn("could not remove last logs timestamp file", zap.Error(err))
	}
}

func lastLogsTimestampFileName(deployment AgentDeployment) string {
	return path.Join(lastLogsTimestampDir(), deployment.ID.String())
}

func lastLogsTimestampDir() string {
	return path.Join(ScratchDir(), "logTimestamp")
}
