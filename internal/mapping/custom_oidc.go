package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/types"
)

func CustomOIDCConfigurationToDTO(
	model types.CustomOIDCConfiguration,
	organizationSlug *string,
	domain string,
) api.CustomOIDCConfiguration {
	return api.CustomOIDCConfiguration{
		ID:                  model.ID,
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
		CustomDomainID:      model.CustomDomainID,
		Name:                model.Name,
		Slug:                model.Slug,
		Enabled:             model.Enabled,
		Issuer:              model.Issuer,
		ClientID:            model.ClientID,
		ClientSecretSet:     model.ClientSecret != "",
		Scopes:              model.Scopes,
		PKCEEnabled:         model.PKCEEnabled,
		SPInitiated:         model.SPInitiated,
		CreateUnknownUsers:  model.CreateUnknownUsers,
		DefaultUserRole:     model.DefaultUserRole,
		AllowedEmailDomains: model.AllowedEmailDomains,
		CallbackURL:         oidc.CustomCallbackURL(domain, organizationSlug, model.Slug),
	}
}
