package types

import (
	"time"

	"github.com/distr-sh/distr/internal/authkey"
	"github.com/google/uuid"
)

type AccessTokenSecretSlot int

const (
	AccessTokenSecretSlot1 AccessTokenSecretSlot = 1
	AccessTokenSecretSlot2 AccessTokenSecretSlot = 2
)

var AccessTokenSecretSlots = []AccessTokenSecretSlot{AccessTokenSecretSlot1, AccessTokenSecretSlot2}

func (slot AccessTokenSecretSlot) Valid() bool {
	return slot == AccessTokenSecretSlot1 || slot == AccessTokenSecretSlot2
}

func (slot AccessTokenSecretSlot) Other() AccessTokenSecretSlot {
	if slot == AccessTokenSecretSlot1 {
		return AccessTokenSecretSlot2
	}
	return AccessTokenSecretSlot1
}

type AccessTokenSecret struct {
	Hash      []byte    `db:"hash"`
	CreatedAt time.Time `db:"created_at"`
	// ExpiresAt is fixed when the secret is created: a token is kept alive by adding a secret that
	// expires later, not by moving an expiration that something in circulation already relies on.
	ExpiresAt  *time.Time `db:"expires_at"`
	LastUsedAt *time.Time `db:"last_used_at"`
}

type AccessToken struct {
	ID        uuid.UUID `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	// ExpiresAt and KeyLastUsedAt belong to the key as a credential of its own and are set only
	// while KeyIsCredential is, which no token created after secrets existed ever is.
	ExpiresAt       *time.Time         `db:"expires_at"`
	KeyIsCredential bool               `db:"key_is_credential"`
	KeyLastUsedAt   *time.Time         `db:"key_last_used_at"`
	LastUsedAt      *time.Time         `db:"last_used_at"`
	Label           *string            `db:"label"`
	Key             authkey.Key        `db:"key"`
	Secret1         *AccessTokenSecret `db:"secret_1"`
	Secret2         *AccessTokenSecret `db:"secret_2"`
	UserAccountID   uuid.UUID          `db:"user_account_id"`
	OrganizationID  uuid.UUID          `db:"organization_id"`
	UserRole        *UserRole          `db:"token_user_role"`
}

func (tok AccessToken) Secret(slot AccessTokenSecretSlot) *AccessTokenSecret {
	if slot == AccessTokenSecretSlot1 {
		return tok.Secret1
	}
	return tok.Secret2
}

func (tok AccessToken) HasSecrets() bool {
	return tok.Secret1 != nil || tok.Secret2 != nil
}

// CredentialCount is how many credentials authenticate this token, counting the key itself while it
// is one. Two is the maximum, so that a credential can be replaced without a gap but a token cannot
// accumulate them.
func (tok AccessToken) CredentialCount() int {
	count := 0
	if tok.KeyIsCredential {
		count++
	}
	for _, slot := range AccessTokenSecretSlots {
		if tok.Secret(slot) != nil {
			count++
		}
	}
	return count
}

func (tok AccessToken) FreeSecretSlot() *AccessTokenSecretSlot {
	for _, slot := range AccessTokenSecretSlots {
		if tok.Secret(slot) == nil {
			return &slot
		}
	}
	return nil
}

// AccessTokenRoleAllowed compares the role a token would act under, which for a token without an
// explicit role is the role its owner has in the organization, against the role of the caller
// creating or changing it, so that a credential restricted below its owner cannot hand out more
// than it has.
func AccessTokenRoleAllowed(tokenRole, callerRole *UserRole, membershipRole UserRole) bool {
	if callerRole == nil {
		return false
	}
	effective := membershipRole
	if tokenRole != nil {
		effective = *tokenRole
	}
	return !effective.GreaterThan(*callerRole)
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
