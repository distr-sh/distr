package authinfo

import (
	"testing"

	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwt"
	. "github.com/onsi/gomega"
)

func TestFromUserJWT(t *testing.T) {
	userToken := func(g *WithT, claims map[string]any) jwt.Token {
		builder := jwt.NewBuilder().Claim(jwt.SubjectKey, uuid.NewString())
		for key, value := range claims {
			builder = builder.Claim(key, value)
		}
		token, err := builder.Build()
		g.Expect(err).ToNot(HaveOccurred())
		return token
	}

	t.Run("a session of a custom OIDC provider is scoped to its organization", func(t *testing.T) {
		g := NewWithT(t)
		info, err := FromUserJWT(userToken(g, map[string]any{
			authjwt.CustomOIDCConfigurationIDKey: uuid.NewString(),
		}))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(info.OrganizationScoped()).To(BeTrue())
	})

	t.Run("a claim that is not a configuration ID at all scopes the session as well", func(t *testing.T) {
		g := NewWithT(t)
		info, err := FromUserJWT(userToken(g, map[string]any{
			authjwt.CustomOIDCConfigurationIDKey: map[string]any{"not": "a configuration ID"},
		}))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(info.OrganizationScoped()).To(BeTrue())
	})

	t.Run("a regular login token is not scoped", func(t *testing.T) {
		g := NewWithT(t)
		info, err := FromUserJWT(userToken(g, nil))
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(info.OrganizationScoped()).To(BeFalse())
	})
}
