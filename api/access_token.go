package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type AccessToken struct {
	ID uuid.UUID `json:"id"`
	// KeyID is the beginning of the token, so that a user can tell which of their tokens a client
	// is configured with. It stays the same when the secrets are rotated, and covers only half of
	// the key, since a token is shown in full exactly once.
	KeyID     string    `json:"keyId"`
	CreatedAt time.Time `json:"createdAt"`
	// ExpiresAt is when the token stops working, which is the last of its secrets to expire, since
	// every secret that is still valid authenticates it. It is not settable: an expiration belongs
	// to the secret it was created with.
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
	// ExpiresAt is the expiration of the secret the token is created with, and a token without one
	// never expires. It cannot be changed afterwards, so a token is kept alive by adding a secret
	// that expires later.
	ExpiresAt *time.Time      `json:"expiresAt"`
	Label     *string         `json:"label"`
	UserRole  *types.UserRole `json:"userRole"`
}

func (r CreateAccessTokenRequest) Validate() error {
	return validateAccessTokenExpiresAt(r.ExpiresAt)
}

// PatchAccessTokenRequest carries the only part of a token that is not fixed when it is created.
// An omitted label is left unchanged and an explicit null clears it.
type PatchAccessTokenRequest struct {
	Label Nullable[string] `json:"label"`
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
