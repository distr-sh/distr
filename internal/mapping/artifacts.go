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
		HiddenInCustomerOverview:  a.HiddenInCustomerOverview,
	}
}

func ArtifactsWithDownloadsToAPI(a types.ArtifactWithDownloads) api.ArtifactsResponse {
	return api.ArtifactsResponse{
		ArtifactWithDownloads:    a,
		ImageUrl:                 CreateImageURL(a.ImageID),
		TitleImageUrl:            CreateImageURL(a.TitleImageID),
		HiddenInCustomerOverview: a.HiddenInCustomerOverview,
	}
}
