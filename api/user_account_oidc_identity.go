package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// UserAccountOIDCIdentity is an identity provider account connected to the current user.
// The subject is deliberately not exposed.
type UserAccountOIDCIdentity struct {
	ID          uuid.UUID          `json:"id"`
	CreatedAt   time.Time          `json:"createdAt"`
	Provider    types.OIDCProvider `json:"provider"`
	Issuer      string             `json:"issuer"`
	Email       *string            `json:"email,omitempty"`
	LastLoginAt *time.Time         `json:"lastLoginAt,omitempty"`
}
