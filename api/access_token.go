package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type AccessToken struct {
	ID        uuid.UUID `json:"id"`
	KeyID     string    `json:"keyId"`
	CreatedAt time.Time `json:"createdAt"`
	// Deprecated: Set only for a token that predates secrets. Every other token expires with the
	// secrets that authenticate it.
	ExpiresAt  *time.Time          `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time          `json:"lastUsedAt,omitempty"`
	Label      *string             `json:"label,omitempty"`
	UserRole   *types.UserRole     `json:"userRole,omitempty"`
	Secrets    []AccessTokenSecret `json:"secrets"`
}

type AccessTokenSecret struct {
	Slot       types.AccessTokenSecretSlot `json:"slot"`
	CreatedAt  time.Time                   `json:"createdAt"`
	ExpiresAt  *time.Time                  `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time                  `json:"lastUsedAt,omitempty"`
}

func (obj AccessToken) WithKey(key string) AccessTokenWithKey {
	return AccessTokenWithKey{obj, key}
}

type AccessTokenWithKey struct {
	AccessToken
	Key string `json:"key"`
}

type CreateAccessTokenRequest struct {
	ExpiresAt *time.Time      `json:"expiresAt"`
	Label     *string         `json:"label"`
	UserRole  *types.UserRole `json:"userRole"`
}

func (r CreateAccessTokenRequest) Validate() error {
	return validateAccessTokenExpiresAt(r.ExpiresAt)
}

type PatchAccessTokenRequest struct {
	Label string `json:"label"`
}

type CreateAccessTokenSecretRequest struct {
	ExpiresAt *time.Time `json:"expiresAt"`
}

func (r CreateAccessTokenSecretRequest) Validate() error {
	return validateAccessTokenExpiresAt(r.ExpiresAt)
}

// An expiration that has already passed would create a credential that is dead on arrival, and
// since a secret's expiration cannot be changed, the only way out of it is to create another one.
func validateAccessTokenExpiresAt(expiresAt *time.Time) error {
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return validation.NewValidationFailedError("the expiration date must be in the future")
	}
	return nil
}
