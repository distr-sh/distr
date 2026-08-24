package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/types"
)

func CustomOIDCConfigurationToAPI(
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

func CustomOIDCConfigurationToPortalOIDCProvider(
	model types.CustomOIDCConfiguration,
	organizationSlug string,
) api.PortalOIDCProvider {
	return api.PortalOIDCProvider{
		Name:        model.Name,
		LoginPath:   oidc.CustomLoginPath(organizationSlug, model.Slug),
		SPInitiated: model.SPInitiated,
	}
}

// CustomOIDCConfigurationToInternal applies the request onto an existing model, so that everything the
// request does not carry -- the id, the organization and the stored client secret -- survives an update.
func CustomOIDCConfigurationToInternal(
	request api.CustomOIDCConfigurationRequest,
	model *types.CustomOIDCConfiguration,
) {
	model.CustomDomainID = request.CustomDomainID
	model.Name = request.Name
	model.Slug = request.Slug
	model.Enabled = request.Enabled
	model.Issuer = request.Issuer
	model.ClientID = request.ClientID
	model.Scopes = request.Scopes
	model.PKCEEnabled = request.PKCEEnabled
	model.SPInitiated = request.SPInitiated
	model.CreateUnknownUsers = request.CreateUnknownUsers
	model.DefaultUserRole = request.DefaultUserRole
	model.AllowedEmailDomains = request.AllowedEmailDomains
}
