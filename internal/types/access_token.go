package types

import (
	"errors"
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

var ErrInvalidAccessTokenSecret = errors.New("invalid access token secret")

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
	Salt       []byte     `db:"salt"`
	Hash       []byte     `db:"hash"`
	CreatedAt  time.Time  `db:"created_at"`
	LastUsedAt *time.Time `db:"last_used_at"`
}

type AccessToken struct {
	ID             uuid.UUID          `db:"id"`
	CreatedAt      time.Time          `db:"created_at"`
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

// VerifySecret returns the slot the given secret belongs to, or nil for a token that has no
// secrets at all. Such a token predates them and is the whole credential on its own, which is
// why it is only accepted when no secret is presented for it: as soon as one of its slots is
// filled, the version of it that is in circulation stops working.
func (tok AccessToken) VerifySecret(secret *authkey.Secret) (*AccessTokenSecretSlot, error) {
	if !tok.HasSecrets() {
		if secret != nil {
			return nil, ErrInvalidAccessTokenSecret
		}
		return nil, nil
	} else if secret == nil {
		return nil, ErrInvalidAccessTokenSecret
	}

	var matched *AccessTokenSecretSlot
	for _, slot := range AccessTokenSecretSlots {
		if s := tok.Secret(slot); s != nil && authkey.VerifySecret(s.Salt, s.Hash, *secret) {
			matched = &slot
		}
	}
	if matched == nil {
		return nil, ErrInvalidAccessTokenSecret
	}
	return matched, nil
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
