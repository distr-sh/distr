package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type AccessToken struct {
	ID uuid.UUID `json:"id"`
	// KeyID is the part of the token that identifies it, so that a user can tell which of their
	// tokens a client is configured with. It stays the same when the secrets are rotated.
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

// PatchAccessTokenRequest supports partial updates: omitted fields are left unchanged, and an
// explicit null clears the field, which means no label and the role of the user.
type PatchAccessTokenRequest struct {
	Label    Nullable[string]         `json:"label"`
	UserRole Nullable[types.UserRole] `json:"userRole"`
}

type CreateAccessTokenSecretRequest struct {
	ExpiresAt *time.Time `json:"expiresAt"`
}
