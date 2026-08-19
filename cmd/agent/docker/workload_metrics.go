package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	composeapi "github.com/docker/compose/v5/pkg/api"
	"github.com/moby/moby/api/types/container"
	mobyClient "github.com/moby/moby/client"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// statsConcurrency bounds parallel ContainerStats calls. Each call takes about one second
// because the daemon collects two samples for the CPU delta (IncludePreviousSample), so
// sequential collection would not fit into the collection interval.
const statsConcurrency = 8

func watchWorkloadMetrics(ctx context.Context) {
	logger.Info("starting workload metrics watch")
	tick := time.Tick(30 * time.Second)
	for ctx.Err() == nil {
		select {
		case <-tick:
			reportWorkloadMetrics(ctx)
		case <-ctx.Done():
			logger.Info("stopping to watch workload metrics")
			return
		}
	}
}

func reportWorkloadMetrics(ctx context.Context) {
	deployments, err := GetExistingDeployments()
	if err != nil {
		logger.Error("failed to get existing deployments for workload metrics", zap.Error(err))
		return
	}

	for _, deployment := range deployments {
		var containers []workloadContainer
		var err error
		switch deployment.DockerType {
		case types.DockerTypeCompose:
			containers, err = composeWorkloadContainers(ctx, deployment)
		case types.DockerTypeSwarm:
			containers, err = swarmWorkloadContainers(ctx, deployment)
		default:
			continue
		}
		if err != nil {
			logger.Warn("failed to list containers for workload metrics",
				zap.String("project", deployment.ProjectName), zap.Error(err))
			continue
		}

		workloads := collectContainerMetrics(ctx, containers)
		request := api.AgentDeploymentWorkloadMetricsRequest{Workloads: workloads}
		if err := client.ReportWorkloadMetrics(ctx, deployment.ID, request); err != nil {
			logger.Error("failed to report workload metrics",
				zap.String("project", deployment.ProjectName), zap.Error(err))
		}
	}
}

type workloadContainer struct {
	ID       string
	Workload string
	Name     string
}

func composeWorkloadContainers(ctx context.Context, deployment AgentDeployment) ([]workloadContainer, error) {
	summaries, err := composeService.Ps(ctx, deployment.ProjectName, composeapi.PsOptions{})
	if err != nil {
		return nil, err
	}
	var result []workloadContainer
	for _, summary := range summaries {
		if summary.State == container.StateRunning {
			result = append(result, workloadContainer{ID: summary.ID, Workload: summary.Service, Name: summary.Name})
		}
	}
	return result, nil
}

func swarmWorkloadContainers(ctx context.Context, deployment AgentDeployment) ([]workloadContainer, error) {
	return nil, errors.New("workload metrics are not supported in swarm mode")
}

// collectContainerMetrics fetches stats for the given containers. Containers that fail (e.g.
// because they stopped between listing and stats collection) are logged and skipped so the
// remaining workloads are still reported.
func collectContainerMetrics(ctx context.Context, containers []workloadContainer) []api.DeploymentWorkloadMetric {
	var mutex sync.Mutex
	var result []api.DeploymentWorkloadMetric
	var eg errgroup.Group
	eg.SetLimit(statsConcurrency)
	for _, c := range containers {
		eg.Go(func() error {
			cpuUsageMillis, memoryBytes, err := containerUsage(ctx, c.ID)
			if err != nil {
				logger.Warn("failed to get container stats",
					zap.String("container", c.Name), zap.Error(err))
				return nil
			}
			metric := api.DeploymentWorkloadMetric{
				Workload:       c.Workload,
				Name:           c.Name,
				CPUUsageMillis: cpuUsageMillis,
				MemoryBytes:    memoryBytes,
			}
			// Limits are optional extra data: report the usage even if the inspect fails.
			if cpuLimitMillis, memoryLimitBytes, err := containerLimits(ctx, c.ID); err != nil {
				logger.Warn("failed to get container limits",
					zap.String("container", c.Name), zap.Error(err))
			} else {
				metric.CPULimitMillis = cpuLimitMillis
				metric.MemoryLimitBytes = memoryLimitBytes
			}
			mutex.Lock()
			defer mutex.Unlock()
			result = append(result, metric)
			return nil
		})
	}
	_ = eg.Wait()
	return result
}

// containerLimits reads the configured limits from the container's HostConfig. The limits from
// the stats response cannot be used instead: for containers without a memory limit the daemon
// reports the host's total memory there, making "no limit" indistinguishable from a real limit.
func containerLimits(ctx context.Context, containerID string) (cpuLimitMillis, memoryLimitBytes *int64, err error) {
	result, err := dockerCli.Client().ContainerInspect(ctx, containerID, mobyClient.ContainerInspectOptions{})
	if err != nil {
		return nil, nil, err
	}
	hostConfig := result.Container.HostConfig
	if hostConfig == nil {
		return nil, nil, nil
	}
	if hostConfig.NanoCPUs > 0 {
		cpuLimitMillis = new(hostConfig.NanoCPUs / 1_000_000)
	} else if hostConfig.CPUQuota > 0 && hostConfig.CPUPeriod > 0 {
		cpuLimitMillis = new(hostConfig.CPUQuota * 1000 / hostConfig.CPUPeriod)
	}
	if hostConfig.Memory > 0 {
		memoryLimitBytes = new(hostConfig.Memory)
	}
	return cpuLimitMillis, memoryLimitBytes, nil
}

func containerUsage(ctx context.Context, containerID string) (cpuUsageMillis, memoryBytes int64, err error) {
	resp, err := dockerCli.Client().ContainerStats(ctx, containerID, mobyClient.ContainerStatsOptions{
		Stream:                false,
		IncludePreviousSample: true,
	})
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return 0, 0, err
	}
	return calculateCPUUsageMillis(stats.PreCPUStats, stats.CPUStats), calculateMemoryUsageBytes(stats.MemoryStats), nil
}

// calculateCPUUsageMillis replicates the CPU calculation of "docker stats"
// (docker/cli cli/command/container/stats_helpers.go), scaled to millicores
// instead of percent so it is comparable with kubernetes pod metrics.
func calculateCPUUsageMillis(previous, current container.CPUStats) int64 {
	cpuDelta := float64(current.CPUUsage.TotalUsage) - float64(previous.CPUUsage.TotalUsage)
	systemDelta := float64(current.SystemUsage) - float64(previous.SystemUsage)
	onlineCPUs := float64(current.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(current.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0 && cpuDelta > 0 {
		return int64(cpuDelta / systemDelta * onlineCPUs * 1000)
	}
	return 0
}

// calculateMemoryUsageBytes replicates the memory calculation of "docker stats": usage
// without the page cache, which is consistent with cadvisor and kubernetes working set.
func calculateMemoryUsageBytes(memory container.MemoryStats) int64 {
	// cgroup v1
	if v, isCgroup1 := memory.Stats["total_inactive_file"]; isCgroup1 && v < memory.Usage {
		return int64(memory.Usage - v)
	}
	// cgroup v2
	if v := memory.Stats["inactive_file"]; v < memory.Usage {
		return int64(memory.Usage - v)
	}
	return int64(memory.Usage)
}
