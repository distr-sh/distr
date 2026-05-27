package types

import (
	"testing"

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
