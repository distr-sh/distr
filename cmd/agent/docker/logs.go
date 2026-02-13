package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"time"

	"github.com/distr-sh/distr/internal/deploymentlogs"
	"github.com/distr-sh/distr/internal/types"
	dockercommand "github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/compose/convert"
	composeapi "github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/stdcopy"
	"go.uber.org/zap"
)

type logsWatcher struct {
	dockerCli      dockercommand.Cli
	composeService composeapi.Compose
	logsExporter   deploymentlogs.Exporter
}

func NewLogsWatcher() *logsWatcher {
	return &logsWatcher{
		dockerCli:      dockerCli,
		composeService: compose.NewComposeService(dockerCli),
		logsExporter:   deploymentlogs.ChunkExporter(client, 100),
	}
}

func (lw *logsWatcher) Watch(ctx context.Context, d time.Duration) {
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
	deployments, err := GetExistingDeployments()
	if err != nil {
		logger.Warn("watch logs could not get deployments", zap.Error(err))
		return
	}

	collector := deploymentlogs.NewCollector()

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

		switch d.DockerType {
		case types.DockerTypeCompose:
			logOptions := composeapi.LogOptions{Timestamps: true}
			if since != nil {
				logOptions.Since = since.Format(time.RFC3339Nano)
			}

			toplevelErr = lw.composeService.Logs(ctx, d.ProjectName, &composeCollector{deploymentCollector}, logOptions)
			if toplevelErr != nil {
				logger.Warn("could not get compose logs", zap.Error(toplevelErr))
			}
		case types.DockerTypeSwarm:
			// Since there is no "docker stack logs" we have to take a small detour:
			// Getting the list of swarm services for the stack and then getting the logs for each service.
			// Because we are interacting with the API directly, we also have to decode the raw stream into its
			// stdout and stderr components.
			apiClient := lw.dockerCli.Client()
			services, err := apiClient.ServiceList(
				ctx,
				swarm.ServiceListOptions{
					Filters: filters.NewArgs(filters.Arg("label", convert.LabelNamespace+"="+d.ProjectName)),
				},
			)
			if err != nil {
				logger.Warn("could not get services for docker stack", zap.Error(err))
				toplevelErr = err
			} else {
				for _, svc := range services {
					// fake closure to close the ReadCloser returned by ServiceLogs after each iteration
					err := func() error {
						logOptions := container.LogsOptions{Timestamps: true, ShowStdout: true, ShowStderr: true}
						if since != nil {
							logOptions.Since = since.Format(time.RFC3339Nano)
						}
						rc, err := apiClient.ServiceLogs(ctx, svc.ID, logOptions)
						if err != nil {
							return err
						}
						defer rc.Close()
						return decodeServiceLogs(svc.Spec.Name, rc, deploymentCollector)
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

	if err := lw.logsExporter.ExportDeploymentLogs(ctx, collector.LogRecords()); err != nil {
		logger.Warn("error exporting logs", zap.Error(err))
	}
}

type composeCollector struct {
	deploymentlogs.DeploymentCollector
}

// Err implements api.LogConsumer.
func (cc *composeCollector) Err(containerName string, message string) {
	cc.AppendMessage(containerName, "Err", message)
}

// Log implements api.LogConsumer.
func (cc *composeCollector) Log(containerName string, message string) {
	cc.AppendMessage(containerName, "Log", message)
}

// Register implements api.LogConsumer.
//
// Noop for now
func (*composeCollector) Register(containerName string) {}

// Status implements api.LogConsumer.
//
// Noop for now
func (*composeCollector) Status(containerName string, message string) {}

func decodeServiceLogs(resource string, r io.Reader, consumer deploymentlogs.DeploymentCollector) error {
	// The docker api provides a multipexed stream for logs which must be demuxed. StdCopy does that.
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&outBuf, &errBuf, r); err != nil {
		return err
	}
	collectFunc := func(name string) func(r, m string) {
		return func(r, m string) { consumer.AppendMessage(r, name, m) }
	}
	streams := []struct {
		*bufio.Scanner
		Collect func(r, m string)
	}{
		{bufio.NewScanner(&outBuf), collectFunc("stdout")},
		{bufio.NewScanner(&errBuf), collectFunc("stderr")},
	}
	for _, stream := range streams {
		for stream.Scan() {
			stream.Collect(resource, stream.Text())
		}
		if stream.Err() != nil {
			return stream.Err()
		}
	}
	return nil
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
