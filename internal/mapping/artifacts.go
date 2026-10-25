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
		HiddenInHome:              a.HiddenInHome,
	}
}

func ArtifactsWithDownloadsToAPI(a types.ArtifactWithDownloads) api.ArtifactsResponse {
	return api.ArtifactsResponse{
		ArtifactWithDownloads: a,
		ImageUrl:              CreateImageURL(a.ImageID),
		TitleImageUrl:         CreateImageURL(a.TitleImageID),
		HiddenInHome:          a.HiddenInHome,
	}
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
