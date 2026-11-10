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

// DeploymentTargetToAPI returns a mapper that shows the deployments of a target as DeploymentToAPI does.
func DeploymentTargetToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(types.DeploymentTargetFull) api.DeploymentTargetResponse {
	deploymentToAPI := DeploymentToAPI(viewerCustomerOrgID, viewerPartnerOrgID)
	return func(dt types.DeploymentTargetFull) api.DeploymentTargetResponse {
		return api.DeploymentTargetResponse{
			ID:                          dt.ID,
			CreatedAt:                   dt.CreatedAt,
			Name:                        dt.Name,
			Type:                        dt.Type,
			ReconnectPending:            dt.ReconnectPending,
			Namespace:                   dt.Namespace,
			Scope:                       dt.Scope,
			CustomerOrganizationID:      dt.CustomerOrganizationID,
			ReportedControllerVersionID: dt.ReportedControllerVersionID,
			MetricsEnabled:              dt.MetricsEnabled,
			ImageCleanupEnabled:         dt.ImageCleanupEnabled,
			AutohealEnabled:             dt.AutohealEnabled,
			AutomaticUpdatesEnabled:     dt.AutomaticUpdatesEnabled,
			DeploymentLogsEnabled:       dt.DeploymentLogsEnabled,
			DeploymentLogsAfter:         dt.DeploymentLogsAfter,
			Resources:                   PtrOrNil(dt.Resources, DeploymentTargetResourcesToAPI),
			DockerEndpoint:              dt.DockerEndpoint,
			CustomerOrganization:        PtrOrNil(dt.CustomerOrganization, CustomerOrganizationToAPI),
			Deployments:                 List(dt.Deployments, deploymentToAPI),
			ControllerVersion:           ControllerVersionToAPI(dt.ControllerVersion),
		}
	}
}

// DeploymentToAPI returns a mapper that shows the creator of the latest revision of a deployment as
// DeploymentRevisionToAPI does.
func DeploymentToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(types.DeploymentWithLatestRevision) api.DeploymentResponse {
	applicationToAPI := ApplicationToAPI(viewerCustomerOrgID, viewerPartnerOrgID)
	creatorToAPI := deploymentRevisionCreatorToAPI(viewerCustomerOrgID, viewerPartnerOrgID)
	return func(d types.DeploymentWithLatestRevision) api.DeploymentResponse {
		return api.DeploymentResponse{
			ID:                                 d.ID,
			CreatedAt:                          d.CreatedAt,
			DeploymentTargetID:                 d.DeploymentTargetID,
			ReleaseName:                        d.ReleaseName,
			ApplicationEntitlementID:           d.ApplicationEntitlementID,
			DockerType:                         d.DockerType,
			AutomaticApplicationUpdatesEnabled: d.AutomaticApplicationUpdatesEnabled,
			DeploymentRevisionID:               d.DeploymentRevisionID,
			DeploymentRevisionCreatedAt:        d.DeploymentRevisionCreatedAt,
			DeploymentRevisionCreatedBy:        creatorToAPI(d.DeploymentRevisionCreatedBy),
			ApplicationID:                      d.ApplicationID,   //nolint:staticcheck
			ApplicationName:                    d.ApplicationName, //nolint:staticcheck
			Application:                        applicationToAPI(d.Application),
			ApplicationVersionID:               d.ApplicationVersionID,
			ApplicationVersionName:             d.ApplicationVersionName,
			ApplicationLink:                    d.ApplicationLink,
			ValuesYaml:                         d.ValuesYaml,
			EnvFileData:                        d.EnvFileData,
			LatestStatus:                       PtrOrNil(d.LatestStatus, DeploymentRevisionStatusToAPI),
			ForceRestart:                       d.ForceRestart,
			IgnoreRevisionSkew:                 d.IgnoreRevisionSkew,
			HelmOptions:                        PtrOrNil(d.HelmOptions, HelmOptionsToAPI),
			CurrentDeploymentRevisionID:        d.CurrentDeploymentRevisionID,
			CurrentStatus:                      PtrOrNil(d.CurrentStatus, DeploymentRevisionStatusToAPI),
			CurrentApplicationVersionID:        d.CurrentApplicationVersionID,
			CurrentApplicationVersionName:      d.CurrentApplicationVersionName,
		}
	}
}

func HelmOptionsToAPI(options types.HelmOptions) api.HelmOptions {
	return api.HelmOptions{
		Timeout:           options.Timeout,
		WaitStrategy:      options.WaitStrategy,
		RollbackOnFailure: options.RollbackOnFailure,
		CleanupOnFailure:  options.CleanupOnFailure,
		ForceConflicts:    options.ForceConflicts,
	}
}

func ControllerVersionToAPI(version types.ControllerVersion) api.ControllerVersion {
	return api.ControllerVersion{
		ID:        version.ID,
		CreatedAt: version.CreatedAt,
		Name:      version.Name,
	}
}

func DeploymentTargetResourcesToAPI(resources types.DeploymentTargetResources) api.DeploymentTargetResources {
	return api.DeploymentTargetResources{
		CPURequest:    resources.CPURequest,
		MemoryRequest: resources.MemoryRequest,
		CPULimit:      resources.CPULimit,
		MemoryLimit:   resources.MemoryLimit,
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
