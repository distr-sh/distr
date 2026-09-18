package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func ApplicationToAPI(a types.Application) api.ApplicationResponse {
	return api.ApplicationResponse{
		Application: a,
		ImageUrl:    CreateImageURL(a.ImageID),
	}
}

func CreateApplicationToInternal(request api.CreateApplicationRequest) types.Application {
	return types.Application{
		Name:                  request.Name,
		Type:                  request.Type,
		VersioningStrategy:    request.VersioningStrategy,
		AllowAutomaticUpdates: request.AllowAutomaticUpdates,
	}
}

func UpdateApplicationToInternal(request api.UpdateApplicationRequest, existing types.Application) types.Application {
	existing.Name = request.Name
	existing.VersioningStrategy = request.VersioningStrategy
	if request.AllowAutomaticUpdates != nil {
		existing.AllowAutomaticUpdates = *request.AllowAutomaticUpdates
	}
	return existing
}
