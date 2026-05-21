package api

import (
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

type ServiceAccountResponse struct {
	ID                     uuid.UUID         `json:"id"`
	CreatedAt              time.Time         `json:"createdAt"`
	Name                   string            `json:"name"`
	AccountRole            types.AccountRole `json:"accountRole"`
	CustomerOrganizationID *uuid.UUID        `json:"customerOrganizationId,omitempty"`
}

type CreateServiceAccountRequest struct {
	Name                   string            `json:"name"`
	AccountRole            types.AccountRole `json:"accountRole"`
	CustomerOrganizationID *uuid.UUID        `json:"customerOrganizationId,omitempty"`
}

type PatchServiceAccountRequest struct {
	Name        *string            `json:"name"`
	AccountRole *types.AccountRole `json:"accountRole"`
}

type ServiceAccountIDRequest struct {
	ServiceAccountID string `json:"-" path:"serviceAccountId"`
}

type ServiceAccountAccessTokenIDRequest struct {
	ServiceAccountID string `json:"-" path:"serviceAccountId"`
	TokenID          string `json:"-" path:"tokenId"`
}

type CreateServiceAccountAccessTokenRequest struct {
	ExpiresAt *time.Time `json:"expiresAt"`
	Label     *string    `json:"label"`
}
