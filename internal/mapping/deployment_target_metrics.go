package mapping

import (
	"cmp"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

//nolint:staticcheck // controllers released before the rename send only the deprecated fields
func DeploymentTargetMetricsRequestToInternal(
	deploymentTargetID uuid.UUID,
	req api.ControllerDeploymentTargetMetricsRequest,
) types.DeploymentTargetMetrics {
	return types.DeploymentTargetMetrics{
		DeploymentTargetID:       deploymentTargetID,
		CPUCoresMillis:           req.CPUCoresMillis,
		CPUUsage:                 req.CPUUsage,
		MemoryBytes:              req.MemoryBytes,
		MemoryUsage:              req.MemoryUsage,
		ControllerCPUUsageMillis: cmp.Or(req.ControllerCPUUsageMillis, req.AgentCPUUsageMillis),
		ControllerMemoryBytes:    cmp.Or(req.ControllerMemoryBytes, req.AgentMemoryBytes),
		ControllerLogBytes:       cmp.Or(req.ControllerLogBytes, req.AgentLogBytes),
		ImageBytes:               req.ImageBytes,
		DiskMetrics:              List(req.DiskMetrics, DeploymentTargetDiskMetricToInternal),
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
		DeploymentTargetID:       metrics.DeploymentTargetID,
		CreatedAt:                metrics.CreatedAt,
		CPUCoresMillis:           metrics.CPUCoresMillis,
		CPUUsage:                 metrics.CPUUsage,
		MemoryBytes:              metrics.MemoryBytes,
		MemoryUsage:              metrics.MemoryUsage,
		ControllerCPUUsageMillis: metrics.ControllerCPUUsageMillis,
		ControllerMemoryBytes:    metrics.ControllerMemoryBytes,
		ControllerLogBytes:       metrics.ControllerLogBytes,
		ImageBytes:               metrics.ImageBytes,
		DiskMetrics:              List(metrics.DiskMetrics, DeploymentTargetDiskMetricToAPI),
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
