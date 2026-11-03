package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func ArtifactToAPI(a types.ArtifactWithTaggedVersion) api.ArtifactResponse {
	return api.ArtifactResponse{
		ArtifactWithTaggedVersion: a,
		ImageUrl:                  CreateImageURL(a.ImageID),
	}
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
	}
}
