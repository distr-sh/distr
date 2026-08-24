package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func UserAccountOIDCIdentityToAPI(
	model types.UserAccountOIDCIdentityWithConfiguration,
) api.UserAccountOIDCIdentity {
	return api.UserAccountOIDCIdentity{
		ID:                model.ID,
		CreatedAt:         model.CreatedAt,
		Provider:          model.Provider,
		Issuer:            model.Issuer,
		Email:             model.Email,
		LastLoginAt:       model.LastLoginAt,
		ConfigurationName: model.ConfigurationName,
		OrganizationName:  model.OrganizationName,
	}
}
