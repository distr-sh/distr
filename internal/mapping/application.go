package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// ApplicationToAPI returns a mapper that converts an Application to its API representation. The
// user who created a version is only included for a vendor's own users, since customers and
// partners must not learn who works for the vendor.
func ApplicationToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(types.Application) api.ApplicationResponse {
	versionToAPI := ApplicationVersionToAPI(viewerCustomerOrgID, viewerPartnerOrgID)
	return func(a types.Application) api.ApplicationResponse {
		return api.ApplicationResponse{
			ID:                    a.ID,
			CreatedAt:             a.CreatedAt,
			Name:                  a.Name,
			Type:                  a.Type,
			ImageID:               a.ImageID,
			ImageUrl:              CreateImageURL(a.ImageID),
			VersioningStrategy:    a.VersioningStrategy,
			AllowAutomaticUpdates: a.AllowAutomaticUpdates,
			Versions:              List(a.Versions, versionToAPI),
		}
	}
}

func ApplicationVersionToAPI(
	viewerCustomerOrgID *uuid.UUID,
	viewerPartnerOrgID *uuid.UUID,
) func(types.ApplicationVersion) api.ApplicationVersionResponse {
	showCreator := viewerCustomerOrgID == nil && viewerPartnerOrgID == nil
	return func(v types.ApplicationVersion) api.ApplicationVersionResponse {
		response := api.ApplicationVersionResponse{
			ID:            v.ID,
			CreatedAt:     v.CreatedAt,
			ArchivedAt:    v.ArchivedAt,
			Name:          v.Name,
			LinkTemplate:  v.LinkTemplate,
			ApplicationID: v.ApplicationID,
			ChartType:     v.ChartType,
			ChartName:     v.ChartName,
			ChartUrl:      v.ChartUrl,
			ChartVersion:  v.ChartVersion,
		}
		if showCreator && v.CreatedByUserAccountID != nil {
			creator := &api.ApplicationVersionCreator{
				ID:      *v.CreatedByUserAccountID,
				ImageID: v.CreatedByImageID,
			}
			if v.CreatedByName != nil {
				creator.Name = *v.CreatedByName
			}
			if v.CreatedByEmail != nil {
				creator.Email = *v.CreatedByEmail
			}
			response.CreatedBy = creator
		}
		return response
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

func CreateApplicationVersionToInternal(request api.CreateApplicationVersionRequest) types.ApplicationVersion {
	return types.ApplicationVersion{
		Name:         request.Name,
		LinkTemplate: request.LinkTemplate,
		ChartType:    request.ChartType,
		ChartName:    request.ChartName,
		ChartUrl:     request.ChartUrl,
		ChartVersion: request.ChartVersion,
		Resources:    List(request.Resources, ApplicationVersionResourceToInternal),
	}
}

func ApplicationVersionResourceToInternal(
	request api.ApplicationVersionResourceRequest,
) types.ApplicationVersionResource {
	return types.ApplicationVersionResource{
		Name:               request.Name,
		Content:            request.Content,
		VisibleToCustomers: request.VisibleToCustomers,
	}
}

func UpdateApplicationVersionToInternal(
	request api.UpdateApplicationVersionRequest,
	existing types.ApplicationVersion,
) types.ApplicationVersion {
	existing.Name = request.Name
	existing.ArchivedAt = request.ArchivedAt
	return existing
}

func UpdateApplicationToInternal(request api.UpdateApplicationRequest, existing types.Application) types.Application {
	existing.Name = request.Name
	existing.VersioningStrategy = request.VersioningStrategy
	if request.AllowAutomaticUpdates != nil {
		existing.AllowAutomaticUpdates = *request.AllowAutomaticUpdates
	}
	return existing
}
