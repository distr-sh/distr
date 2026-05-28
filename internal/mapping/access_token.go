package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func AccessTokenToDTO(model types.AccessToken) api.AccessToken {
	return api.AccessToken{
		ID:         model.ID,
		CreatedAt:  model.CreatedAt,
		ExpiresAt:  model.ExpiresAt,
		LastUsedAt: model.LastUsedAt,
		Label:      model.Label,
		UserRole:   model.UserRole,
	}
}
