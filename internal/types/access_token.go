package types

import (
	"time"

	"github.com/distr-sh/distr/internal/authkey"
	"github.com/google/uuid"
)

type AccessToken struct {
	ID             uuid.UUID   `db:"id"`
	CreatedAt      time.Time   `db:"created_at"`
	ExpiresAt      *time.Time  `db:"expires_at"`
	LastUsedAt     *time.Time  `db:"last_used_at"`
	Label          *string     `db:"label"`
	Key            authkey.Key `db:"key"`
	UserAccountID  uuid.UUID   `db:"user_account_id"`
	OrganizationID uuid.UUID   `db:"organization_id"`
	UserRole       *UserRole   `db:"token_user_role"`
}

func (tok AccessToken) HasExpired() bool {
	return tok.ExpiresAt == nil || tok.ExpiresAt.After(time.Now())
}

type AccessTokenWithUserAccount struct {
	AccessToken
	UserAccount            UserAccount `db:"user_account"`
	UserRole               UserRole    `db:"user_role"`
	CustomerOrganizationID *uuid.UUID  `db:"customer_organization_id"`
}

// EffectiveUserRole returns the role this token may act under, capped at the
// user's current role in the organization. If the token does not have an
// explicit role, the user's current role is used as-is. The cap is re-applied
// on every authenticated request, so demoting the user automatically lowers
// the effective role of all their existing tokens.
func (tok AccessTokenWithUserAccount) EffectiveUserRole() UserRole {
	if tok.AccessToken.UserRole != nil && !tok.AccessToken.UserRole.GreaterThan(tok.UserRole) {
		return *tok.AccessToken.UserRole
	}
	return tok.UserRole
}
