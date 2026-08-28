package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// AdvisoryToAPI returns a mapper that withholds the creator from customer and partner viewers,
// the same way DeploymentRevisionToAPI does.
func AdvisoryToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(types.AdvisoryWithDetails) api.Advisory {
	showCreator := viewerCustomerOrgID == nil && viewerPartnerOrgID == nil
	return func(advisory types.AdvisoryWithDetails) api.Advisory {
		result := api.Advisory{
			ID:                   advisory.ID,
			CreatedAt:            advisory.CreatedAt,
			UpdatedAt:            advisory.UpdatedAt,
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
			Affected:             advisory.CallerAffected,
		}
		if showCreator {
			result.CreatedByUserName = advisory.CreatedByUserName
			result.CreatedByImageURL = CreateImageURL(advisory.CreatedByImageID)
		}
		return result
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
		ApplicationType:        version.ApplicationType,
		ApplicationImageURL:    CreateImageURL(version.ApplicationImageID),
		ApplicationVersionID:   version.ApplicationVersionID,
		ApplicationVersionName: version.ApplicationVersionName,
		Relation:               version.Relation,
	}
}

func AdvisoryArtifactVersionToAPI(
	version types.AdvisoryArtifactVersion,
) api.AdvisoryArtifactVersion {
	return api.AdvisoryArtifactVersion{
		ArtifactID:            version.ArtifactID,
		ArtifactName:          version.ArtifactName,
		ArtifactImageURL:      CreateImageURL(version.ArtifactImageID),
		ArtifactVersionID:     version.ArtifactVersionID,
		ArtifactVersionName:   version.ArtifactVersionName,
		ArtifactVersionDigest: version.ArtifactVersionDigest,
		ArtifactVersionTags:   version.ArtifactVersionTags,
		Relation:              version.Relation,
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
