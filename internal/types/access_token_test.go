package types

import (
	"testing"

	"github.com/distr-sh/distr/internal/authkey"
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

func TestVerifySecret(t *testing.T) {
	g := NewWithT(t)

	newSecret := func() (authkey.Secret, AccessTokenSecret) {
		secret, err := authkey.NewSecret()
		g.Expect(err).ToNot(HaveOccurred())
		salt, err := authkey.NewSalt()
		g.Expect(err).ToNot(HaveOccurred())
		return secret, AccessTokenSecret{Salt: salt, Hash: secret.Hash(salt)}
	}

	first, firstStored := newSecret()
	second, secondStored := newSecret()
	unknown, _ := newSecret()

	// A token without secrets predates them and authenticates on its key alone, but only when
	// no secret is presented for it.
	legacy := AccessToken{}
	slot, err := legacy.VerifySecret(nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(slot).To(BeNil())
	_, err = legacy.VerifySecret(&first)
	g.Expect(err).To(MatchError(ErrInvalidAccessTokenSecret))

	// As soon as a token has a secret, presenting none is no longer enough.
	secured := AccessToken{Secret1: &firstStored}
	_, err = secured.VerifySecret(nil)
	g.Expect(err).To(MatchError(ErrInvalidAccessTokenSecret))
	slot, err = secured.VerifySecret(&first)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(slot).To(HaveValue(Equal(AccessTokenSecretSlot1)))
	_, err = secured.VerifySecret(&unknown)
	g.Expect(err).To(MatchError(ErrInvalidAccessTokenSecret))

	// Both slots are equal peers, which is what makes rotation without downtime possible.
	rotating := AccessToken{Secret1: &firstStored, Secret2: &secondStored}
	slot, err = rotating.VerifySecret(&second)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(slot).To(HaveValue(Equal(AccessTokenSecretSlot2)))
	slot, err = rotating.VerifySecret(&first)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(slot).To(HaveValue(Equal(AccessTokenSecretSlot1)))

	// A secret only authenticates the token it was created for.
	g.Expect(AccessToken{Secret2: &secondStored}.VerifySecret(&first)).
		Error().To(MatchError(ErrInvalidAccessTokenSecret))
}
