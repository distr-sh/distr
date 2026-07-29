package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func AdvisoryToAPI(advisory types.AdvisoryWithDetails) api.Advisory {
	return api.Advisory{
		ID:                   advisory.ID,
		CreatedAt:            advisory.CreatedAt,
		UpdatedAt:            advisory.UpdatedAt,
		CreatedByUserName:    advisory.CreatedByUserName,
		CreatedByImageURL:    CreateImageURL(advisory.CreatedByImageID),
		Title:                advisory.Title,
		Status:               advisory.Status,
		Severity:             advisory.Severity,
		CveID:                advisory.CveID,
		Tags:                 advisory.Tags,
		AffectedVersionCount: advisory.AffectedVersionCount,
		FixedVersionCount:    advisory.FixedVersionCount,
		ReferenceCount:       advisory.ReferenceCount,
		PublishedAt:          advisory.PublishedAt,
		ResolvedAt:           advisory.ResolvedAt,
	}
}

func AdvisoryReferenceToAPI(reference types.AdvisoryReference) api.AdvisoryReference {
	return api.AdvisoryReference{
		URL:   reference.URL,
		Label: reference.Label,
	}
}

func AdvisoryApplicationVersionToAPI(
	version types.AdvisoryApplicationVersion,
) api.AdvisoryApplicationVersion {
	return api.AdvisoryApplicationVersion{
		ApplicationID:          version.ApplicationID,
		ApplicationName:        version.ApplicationName,
		ApplicationVersionID:   version.ApplicationVersionID,
		ApplicationVersionName: version.ApplicationVersionName,
		Relation:               version.Relation,
	}
}

func AdvisoryArtifactVersionToAPI(
	version types.AdvisoryArtifactVersion,
) api.AdvisoryArtifactVersion {
	return api.AdvisoryArtifactVersion{
		ArtifactID:          version.ArtifactID,
		ArtifactName:        version.ArtifactName,
		ArtifactVersionID:   version.ArtifactVersionID,
		ArtifactVersionName: version.ArtifactVersionName,
		Relation:            version.Relation,
	}
}

func AdvisoryEventToAPI(event types.AdvisoryEventWithUser) api.AdvisoryEvent {
	return api.AdvisoryEvent{
		ID:           event.ID,
		CreatedAt:    event.CreatedAt,
		Type:         event.Type,
		Message:      event.Message,
		UserName:     event.UserName,
		UserImageURL: CreateImageURL(event.UserImageID),
	}
}

func AdvisoryImpactedDeploymentToAPI(
	deployment types.AdvisoryImpactedDeployment,
) api.AdvisoryImpactedDeployment {
	return api.AdvisoryImpactedDeployment{
		CustomerOrganizationID:        deployment.CustomerOrganizationID,
		CustomerOrganizationName:      deployment.CustomerOrganizationName,
		DeploymentID:                  deployment.DeploymentID,
		DeploymentTargetID:            deployment.DeploymentTargetID,
		DeploymentTargetName:          deployment.DeploymentTargetName,
		ApplicationID:                 deployment.ApplicationID,
		ApplicationName:               deployment.ApplicationName,
		ApplicationVersionID:          deployment.ApplicationVersionID,
		ApplicationVersionName:        deployment.ApplicationVersionName,
		CurrentApplicationVersionID:   deployment.CurrentApplicationVersionID,
		CurrentApplicationVersionName: deployment.CurrentApplicationVersionName,
		State:                         deployment.State,
		LastDeployedAt:                deployment.LastDeployedAt,
	}
}

func AdvisoryImpactedPullToAPI(pull types.AdvisoryImpactedPull) api.AdvisoryImpactedPull {
	return api.AdvisoryImpactedPull{
		CustomerOrganizationID:   pull.CustomerOrganizationID,
		CustomerOrganizationName: pull.CustomerOrganizationName,
		ArtifactID:               pull.ArtifactID,
		ArtifactName:             pull.ArtifactName,
		ArtifactVersionID:        pull.ArtifactVersionID,
		ArtifactVersionName:      pull.ArtifactVersionName,
		PullCount:                pull.PullCount,
		LastPulledAt:             pull.LastPulledAt,
	}
}
