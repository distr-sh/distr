package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func DeploymentResourceMetricsRequestToInternal(
	deploymentID uuid.UUID,
	req api.AgentDeploymentResourceMetricsRequest,
) types.DeploymentMetrics {
	return types.DeploymentMetrics{
		DeploymentID: deploymentID,
		Resources:    List(req.Resources, DeploymentResourceMetricToInternal),
	}
}

func DeploymentResourceMetricToInternal(resource api.DeploymentResourceMetric) types.DeploymentResourceMetric {
	return types.DeploymentResourceMetric{
		Resource:         resource.Resource,
		Container:        resource.Container,
		CPUUsageMillis:   resource.CPUUsageMillis,
		MemoryBytes:      resource.MemoryBytes,
		CPULimitMillis:   resource.CPULimitMillis,
		MemoryLimitBytes: resource.MemoryLimitBytes,
	}
}

func DeploymentMetricsToAPI(metrics types.DeploymentMetrics) api.DeploymentResourceMetrics {
	return api.DeploymentResourceMetrics{
		DeploymentID: metrics.DeploymentID,
		CreatedAt:    metrics.CreatedAt,
		Resources:    List(metrics.Resources, DeploymentResourceMetricToAPI),
	}
}

func DeploymentResourceMetricToAPI(resource types.DeploymentResourceMetric) api.DeploymentResourceMetric {
	return api.DeploymentResourceMetric{
		Resource:         resource.Resource,
		Container:        resource.Container,
		CPUUsageMillis:   resource.CPUUsageMillis,
		MemoryBytes:      resource.MemoryBytes,
		CPULimitMillis:   resource.CPULimitMillis,
		MemoryLimitBytes: resource.MemoryLimitBytes,
	}
}
