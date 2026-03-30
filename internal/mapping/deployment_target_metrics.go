package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func DeploymentTargetMetricsRequestToInternal(
	deploymentTargetID uuid.UUID,
	req api.AgentDeploymentTargetMetricsRequest,
) types.DeploymentTargetMetrics {
	return types.DeploymentTargetMetrics{
		DeploymentTargetID: deploymentTargetID,
		CPUCoresMillis:     req.CPUCoresMillis,
		CPUUsage:           req.CPUUsage,
		MemoryBytes:        req.MemoryBytes,
		MemoryUsage:        req.MemoryUsage,
		DiskMetrics:        List(req.DiskMetrics, DeploymentTargetDiskMetricToInternal),
	}
}

func DeploymentTargetDiskMetricToInternal(disk api.DeploymentTargetDiskMetric) types.DeploymentTargetDiskMetric {
	return types.DeploymentTargetDiskMetric{
		Device:     disk.Device,
		Path:       disk.Path,
		FsType:     disk.FsType,
		BytesTotal: disk.BytesTotal,
		BytesUsed:  disk.BytesUsed,
	}
}

func DeploymentTargetMetricsToAPI(metrics types.DeploymentTargetMetrics) api.DeploymentTargetMetrics {
	return api.DeploymentTargetMetrics{
		DeploymentTargetID: metrics.DeploymentTargetID,
		CPUCoresMillis:     metrics.CPUCoresMillis,
		CPUUsage:           metrics.CPUUsage,
		MemoryBytes:        metrics.MemoryBytes,
		MemoryUsage:        metrics.MemoryUsage,
		DiskMetrics:        List(metrics.DiskMetrics, DeploymentTargetDiskMetricToAPI),
	}
}

func DeploymentTargetDiskMetricToAPI(disk types.DeploymentTargetDiskMetric) api.DeploymentTargetDiskMetric {
	return api.DeploymentTargetDiskMetric{
		Device:     disk.Device,
		Path:       disk.Path,
		FsType:     disk.FsType,
		BytesTotal: disk.BytesTotal,
		BytesUsed:  disk.BytesUsed,
	}
}
