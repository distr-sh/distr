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
	// ConfigurationName and OrganizationName are set for provider "custom": the display name of the
	// configuration and the organization that controls it. Naming the organization matters, because
	// such a login is governed by that organization rather than by Distr.
	ConfigurationName *string `json:"configurationName,omitempty"`
	OrganizationName  *string `json:"organizationName,omitempty"`
}
