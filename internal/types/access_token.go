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
	// ExpiresAt is the expiration of a token that predates secrets. A token with secrets expires
	// with them, so nothing writes this column anymore.
	ExpiresAt      *time.Time         `db:"expires_at"`
	LastUsedAt     *time.Time         `db:"last_used_at"`
	Label          *string            `db:"label"`
	Key            authkey.Key        `db:"key"`
	Secret1        *AccessTokenSecret `db:"secret_1"`
	Secret2        *AccessTokenSecret `db:"secret_2"`
	UserAccountID  uuid.UUID          `db:"user_account_id"`
	OrganizationID uuid.UUID          `db:"organization_id"`
	UserRole       *UserRole          `db:"token_user_role"`
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

// EffectiveExpiresAt is when the token stops working, which for a token with secrets is the last of
// them to expire, since every secret that is still valid authenticates it. It is nil when the token
// never expires.
func (tok AccessToken) EffectiveExpiresAt() *time.Time {
	if !tok.HasSecrets() {
		return tok.ExpiresAt
	}
	var last *time.Time
	for _, slot := range AccessTokenSecretSlots {
		if secret := tok.Secret(slot); secret != nil {
			if secret.ExpiresAt == nil {
				return nil
			}
			if last == nil || secret.ExpiresAt.After(*last) {
				last = secret.ExpiresAt
			}
		}
	}
	return last
}

// KeyID returns the part of the token that identifies it, in the encoding its owner finds at the
// beginning of the token itself. For a token that predates secrets that is the hex encoding it was
// issued in, and since such a token is nothing but its key, the whole token.
func (tok AccessToken) KeyID() string {
	if !tok.HasSecrets() {
		return authkey.Token{Key: tok.Key}.Serialize()
	}
	return tok.Key.Serialize()
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
