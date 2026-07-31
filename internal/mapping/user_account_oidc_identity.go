package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func UserAccountOIDCIdentityToDTO(model types.UserAccountOIDCIdentity) api.UserAccountOIDCIdentity {
	return api.UserAccountOIDCIdentity{
		ID:          model.ID,
		CreatedAt:   model.CreatedAt,
		Provider:    model.Provider,
		Issuer:      model.Issuer,
		Email:       model.Email,
		LastLoginAt: model.LastLoginAt,
	}
}
