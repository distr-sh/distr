package types

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestEffectiveUserRole(t *testing.T) {
	g := NewWithT(t)

	mk := func(tokenRole *UserRole, orgRole UserRole) AccessTokenWithUserAccount {
		return AccessTokenWithUserAccount{
			AccessToken: AccessToken{UserRole: tokenRole},
			UserRole:    orgRole,
		}
	}

	// Explicit PAT role caps the org role.
	g.Expect(mk(new(UserRoleReadOnly), UserRoleAdmin).EffectiveUserRole()).
		To(Equal(UserRoleReadOnly))
	g.Expect(mk(new(UserRoleReadWrite), UserRoleAdmin).EffectiveUserRole()).
		To(Equal(UserRoleReadWrite))

	// PAT role above org role is clamped to org role (e.g. user demoted after PAT issued).
	g.Expect(mk(new(UserRoleAdmin), UserRoleReadOnly).EffectiveUserRole()).
		To(Equal(UserRoleReadOnly))

	// No PAT role → inherit org role (legacy behavior for pre-migration tokens).
	g.Expect(mk(nil, UserRoleAdmin).EffectiveUserRole()).To(Equal(UserRoleAdmin))
	g.Expect(mk(nil, UserRoleReadOnly).EffectiveUserRole()).To(Equal(UserRoleReadOnly))

	// Equal roles return the role unchanged.
	g.Expect(mk(new(UserRoleReadWrite), UserRoleReadWrite).EffectiveUserRole()).
		To(Equal(UserRoleReadWrite))
}

func TestEffectiveExpiresAt(t *testing.T) {
	g := NewWithT(t)

	early := time.Now().Add(time.Hour)
	late := early.Add(time.Hour)

	// A token that predates secrets carries its own expiration.
	g.Expect(AccessToken{ExpiresAt: &early}.EffectiveExpiresAt()).To(HaveValue(Equal(early)))

	// Every secret that is still valid authenticates the token, so the last one to expire decides,
	// and a secret that never expires keeps the token alive indefinitely.
	g.Expect(AccessToken{
		Secret1: &AccessTokenSecret{ExpiresAt: &early},
		Secret2: &AccessTokenSecret{ExpiresAt: &late},
	}.EffectiveExpiresAt()).To(HaveValue(Equal(late)))
	g.Expect(AccessToken{
		Secret1: &AccessTokenSecret{ExpiresAt: &early},
		Secret2: &AccessTokenSecret{},
	}.EffectiveExpiresAt()).To(BeNil())
}
