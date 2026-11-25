package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func ArtifactToAPI(a types.ArtifactWithTaggedVersion) api.ArtifactResponse {
	return api.ArtifactResponse{
		ArtifactWithTaggedVersion: a,
		ImageUrl:                  CreateImageURL(a.ImageID),
		TitleImageUrl:             CreateImageURL(a.TitleImageID),
		Featured:                  a.Featured,
		VersioningStrategy:        a.VersioningStrategy,
	}
}

func SyncArtifactToAPI(a types.ArtifactWithTaggedVersion, skippedTags []string) api.SyncArtifactResponse {
	return api.SyncArtifactResponse{ArtifactResponse: ArtifactToAPI(a), SkippedTags: skippedTags}
}

// DeleteArtifactTagToAPI takes the artifact that remains after the deletion, nil if the deletion removed it.
func DeleteArtifactTagToAPI(remaining *types.ArtifactWithTaggedVersion) api.DeleteArtifactTagResponse {
	if remaining == nil {
		return api.DeleteArtifactTagResponse{ArtifactDeleted: true}
	}
	return api.DeleteArtifactTagResponse{Artifact: new(ArtifactToAPI(*remaining))}
}

func ArtifactsWithDownloadsToAPI(a types.ArtifactWithDownloads) api.ArtifactsResponse {
	return api.ArtifactsResponse{
		ArtifactWithDownloads: a,
		ImageUrl:              CreateImageURL(a.ImageID),
		TitleImageUrl:         CreateImageURL(a.TitleImageID),
		Featured:              a.Featured,
		VersioningStrategy:    a.VersioningStrategy,
	}
}

func LatestArtifactPullToAPI(update types.CustomerArtifactUpdate) api.LatestArtifactPullResponse {
	result := api.LatestArtifactPullResponse{
		CustomerOrganizationID: update.CustomerOrganizationID,
		ArtifactID:             update.Artifact.ID,
		ArtifactName:           update.Artifact.Name,
		ArtifactImageUrl:       CreateImageURL(update.Artifact.ImageID),
		VersionName:            update.ArtifactVersion.Name,
	}
	if update.AvailableUpdate != nil {
		result.AvailableUpdate = &update.AvailableUpdate.Name
	}
	return result
}

// CustomerArtifactToAPI adds the last pull of the customer organization to the artifact.
func CustomerArtifactToAPI(a types.ArtifactWithDownloads, pull *types.ArtifactVersionPull) api.ArtifactsResponse {
	result := ArtifactsWithDownloadsToAPI(a)
	if pull != nil {
		result.LastPull = &api.ArtifactLastPull{
			VersionName: pull.ArtifactVersion.Name,
			CreatedAt:   pull.CreatedAt,
		}
		if pull.UserAccount != nil {
			result.LastPull.PulledByName = &pull.UserAccount.Name
			result.LastPull.PulledByImageUrl = CreateImageURL(pull.UserAccount.ImageID)
		}
		if pull.DeploymentTarget != nil {
			result.LastPull.DeploymentTargetName = &pull.DeploymentTarget.Name
		}
	}
	return result
}
