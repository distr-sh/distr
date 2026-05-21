package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func ServiceAccountToAPI(sa types.ServiceAccount) api.ServiceAccountResponse {
	return api.ServiceAccountResponse{
		ID:                     sa.ID,
		CreatedAt:              sa.CreatedAt,
		Name:                   sa.Name,
		AccountRole:            sa.AccountRole,
		CustomerOrganizationID: sa.CustomerOrganizationID,
	}
}

func ServiceAccountAccessTokenToDTO(t types.ServiceAccountAccessToken) api.AccessToken {
	return api.AccessToken{
		ID:         t.ID,
		CreatedAt:  t.CreatedAt,
		ExpiresAt:  t.ExpiresAt,
		LastUsedAt: t.LastUsedAt,
		Label:      t.Label,
	}
}
