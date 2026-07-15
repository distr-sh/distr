package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func OrganizationBrandingToInternal(r api.UpsertOrganizationBrandingRequest) types.OrganizationBranding {
	return types.OrganizationBranding{
		Title:          r.Title,
		Description:    r.Description,
		LogoImageID:    r.LogoImageID,
		PageTitle:      r.PageTitle,
		FaviconImageID: r.FaviconImageID,
	}
}

// OrganizationBrandingToPortalResponse maps the host-resolved branding to the public portal response. The logo and
// favicon are public files so the (unauthenticated) login page of a custom app domain can load them directly.
func OrganizationBrandingToPortalResponse(b types.OrganizationBranding) api.PortalResponse {
	return api.PortalResponse{
		PageTitle:  b.PageTitle,
		FaviconUrl: CreatePublicImageURL(b.FaviconImageID),
		LogoUrl:    CreatePublicImageURL(b.LogoImageID),
	}
}
