package mapping

import (
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
)

func AccessTokenToAPI(model types.AccessToken) api.AccessToken {
	return api.AccessToken{
		ID:         model.ID,
		KeyID:      model.KeyID(),
		CreatedAt:  model.CreatedAt,
		ExpiresAt:  model.ExpiresAt,
		LastUsedAt: model.LastUsedAt,
		Label:      model.Label,
		UserRole:   model.UserRole,
		Secrets:    AccessTokenSecretsToAPI(model),
	}
}

func AccessTokenSecretsToAPI(model types.AccessToken) []api.AccessTokenSecret {
	secrets := make([]api.AccessTokenSecret, 0, len(types.AccessTokenSecretSlots))
	for _, slot := range types.AccessTokenSecretSlots {
		if secret := model.Secret(slot); secret != nil {
			secrets = append(secrets, api.AccessTokenSecret{
				Slot:       slot,
				CreatedAt:  secret.CreatedAt,
				LastUsedAt: secret.LastUsedAt,
			})
		}
	}
	return secrets
}
