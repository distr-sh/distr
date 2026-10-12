package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

//nolint:staticcheck // clients written against the response model send only the deprecated field
func CreateDeploymentTargetToInternal(request api.CreateDeploymentTargetRequest) types.DeploymentTargetFull {
	dt := deploymentTargetSettingsToInternal(request.DeploymentTargetSettings)
	dt.Type = request.Type
	dt.Namespace = request.Namespace
	dt.Scope = request.Scope
	dt.AutohealEnabled = request.AutohealEnabled
	dt.CustomerOrganizationID = referencedID(request.CustomerOrganizationID, request.CustomerOrganization)
	return types.DeploymentTargetFull{DeploymentTarget: dt}
}

// UpdateDeploymentTargetToInternal leaves ControllerVersionID nil unless the request names a version, so the
// update keeps the version the target has in the database rather than the one it had when it was read.
//
//nolint:staticcheck // clients written against the response model send only the deprecated field
func UpdateDeploymentTargetToInternal(
	id uuid.UUID,
	request api.UpdateDeploymentTargetRequest,
) types.DeploymentTargetFull {
	dt := deploymentTargetSettingsToInternal(request.DeploymentTargetSettings)
	dt.ID = id
	dt.ControllerVersionID = referencedID(request.ControllerVersionID, request.ControllerVersion)
	return types.DeploymentTargetFull{DeploymentTarget: dt}
}

func referencedID(id *uuid.UUID, deprecated *api.EntityReference) *uuid.UUID {
	if id == nil && deprecated != nil {
		id = &deprecated.ID
	}
	if id == nil || *id == uuid.Nil {
		return nil
	}
	return id
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
