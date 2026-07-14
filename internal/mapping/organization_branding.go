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
