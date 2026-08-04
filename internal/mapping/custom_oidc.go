package mapping

import (
	"fmt"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func CustomOIDCConfigurationToDTO(
	model types.CustomOIDCConfiguration,
	domain string,
) api.CustomOIDCConfiguration {
	return api.CustomOIDCConfiguration{
		ID:                  model.ID,
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
		CustomDomainID:      model.CustomDomainID,
		Name:                model.Name,
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
		CallbackURL:         CustomOIDCCallbackURL(domain, model),
	}
}

func CustomOIDCCallbackURL(domain string, model types.CustomOIDCConfiguration) string {
	if domain == "" {
		return ""
	}
	return fmt.Sprintf("https://%v/api/v1/auth/oidc/custom/%v/callback", domain, model.ID)
}
