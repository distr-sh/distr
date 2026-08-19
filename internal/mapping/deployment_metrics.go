package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func DeploymentWorkloadMetricsRequestToInternal(
	deploymentID uuid.UUID,
	req api.AgentDeploymentWorkloadMetricsRequest,
) types.DeploymentMetrics {
	return types.DeploymentMetrics{
		DeploymentID: deploymentID,
		Workloads:    List(req.Workloads, DeploymentWorkloadMetricToInternal),
	}
}

func DeploymentWorkloadMetricToInternal(workload api.DeploymentWorkloadMetric) types.DeploymentWorkloadMetric {
	return types.DeploymentWorkloadMetric{
		Workload:         workload.Workload,
		Name:             workload.Name,
		CPUUsageMillis:   workload.CPUUsageMillis,
		MemoryBytes:      workload.MemoryBytes,
		CPULimitMillis:   workload.CPULimitMillis,
		MemoryLimitBytes: workload.MemoryLimitBytes,
	}
}

func DeploymentMetricsToAPI(metrics types.DeploymentMetrics) api.DeploymentWorkloadMetrics {
	return api.DeploymentWorkloadMetrics{
		DeploymentID: metrics.DeploymentID,
		CreatedAt:    metrics.CreatedAt,
		Workloads:    List(metrics.Workloads, DeploymentWorkloadMetricToAPI),
	}
}

func DeploymentWorkloadMetricToAPI(workload types.DeploymentWorkloadMetric) api.DeploymentWorkloadMetric {
	return api.DeploymentWorkloadMetric{
		Workload:         workload.Workload,
		Name:             workload.Name,
		CPUUsageMillis:   workload.CPUUsageMillis,
		MemoryBytes:      workload.MemoryBytes,
		CPULimitMillis:   workload.CPULimitMillis,
		MemoryLimitBytes: workload.MemoryLimitBytes,
	}
}
