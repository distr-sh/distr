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
	UserAccount UserAccount `db:"user_account"`
	// UserRole is the role the user currently holds in the token's organization.
	// It is nil when the user is not a member of that organization, which is the
	// case for super-admin-owned tokens (super admins are not org members).
	UserRole               *UserRole  `db:"user_role"`
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id"`
}

// EffectiveUserRole returns the role this token may act under, capped at the
// user's current role in the organization. If the token does not have an
// explicit role, the user's current role is used as-is. The cap is re-applied
// on every authenticated request, so demoting the user automatically lowers
// the effective role of all their existing tokens.
//
// When the user has no membership in the token's organization (e.g. a
// super-admin-owned token), there is no role to cap against and the token's own
// role is authoritative.
//
// Super admins are never organization members and must only ever act under a
// read-only role through a token, regardless of the token's stored role. A nil
// membership role can otherwise only occur for a non-member whose token is
// rejected by DbAuthenticator anyway, so read-only is a safe floor for it too.
func (tok AccessTokenWithUserAccount) EffectiveUserRole() UserRole {
	if tok.UserAccount.IsSuperAdmin || tok.UserRole == nil {
		return UserRoleReadOnly
	}
	if tok.AccessToken.UserRole != nil && !tok.AccessToken.UserRole.GreaterThan(*tok.UserRole) {
		return *tok.AccessToken.UserRole
	}
	return *tok.UserRole
}
