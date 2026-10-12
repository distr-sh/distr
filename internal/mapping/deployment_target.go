package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func CreateDeploymentTargetToInternal(request api.CreateDeploymentTargetRequest) types.DeploymentTargetFull {
	dt := deploymentTargetSettingsToInternal(request.DeploymentTargetSettings)
	dt.Type = request.Type
	dt.Namespace = request.Namespace
	dt.Scope = request.Scope
	dt.AutohealEnabled = request.AutohealEnabled
	return types.DeploymentTargetFull{DeploymentTarget: dt}
}

// UpdateDeploymentTargetToInternal leaves ControllerVersionID nil unless the request names a version, so the
// update keeps the version the target has in the database rather than the one it had when it was read.
func UpdateDeploymentTargetToInternal(
	id uuid.UUID,
	request api.UpdateDeploymentTargetRequest,
) types.DeploymentTargetFull {
	dt := deploymentTargetSettingsToInternal(request.DeploymentTargetSettings)
	dt.ID = id
	if request.ControllerVersion != nil && request.ControllerVersion.ID != uuid.Nil {
		dt.ControllerVersionID = &request.ControllerVersion.ID
	}
	return types.DeploymentTargetFull{DeploymentTarget: dt}
}

func deploymentTargetSettingsToInternal(settings api.DeploymentTargetSettings) types.DeploymentTarget {
	return types.DeploymentTarget{
		Name:                    settings.Name,
		MetricsEnabled:          settings.MetricsEnabled,
		ImageCleanupEnabled:     settings.ImageCleanupEnabled,
		AutomaticUpdatesEnabled: settings.AutomaticUpdatesEnabled,
		DeploymentLogsEnabled:   settings.DeploymentLogsEnabled,
		DeploymentLogsAfter:     settings.DeploymentLogsAfter,
		Resources:               PtrOrNil(settings.Resources, DeploymentTargetResourcesToInternal),
		DockerEndpoint:          settings.DockerEndpoint,
	}
}

func DeploymentTargetResourcesToInternal(resources api.DeploymentTargetResources) types.DeploymentTargetResources {
	return types.DeploymentTargetResources{
		CPURequest:    resources.CPURequest,
		MemoryRequest: resources.MemoryRequest,
		CPULimit:      resources.CPULimit,
		MemoryLimit:   resources.MemoryLimit,
	}
}
