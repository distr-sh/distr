package authjwt

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwk"
	. "github.com/onsi/gomega"
)

func TestGenerateContextTokenKeepsCustomOIDCSession(t *testing.T) {
	g := NewWithT(t)
	signingKey = func() (jwk.SymmetricKey, error) { return jwk.Import[jwk.SymmetricKey]([]byte("test-secret")) }
	user := types.UserAccount{ID: uuid.New()}
	org := types.OrganizationWithUserRole{Organization: types.Organization{ID: uuid.New()}, UserRole: types.UserRoleAdmin}
	customerOrg := org
	customerOrg.CustomerOrganizationID = new(uuid.New())

	login, _, err := GenerateCustomOIDCToken(user, org, uuid.New())
	g.Expect(err).ToNot(HaveOccurred())
	inLens, _, err := GenerateContextToken(login, user, customerOrg, true)
	g.Expect(err).ToNot(HaveOccurred())
	leftLens, _, err := GenerateContextToken(inLens, user, org, false)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(inLens.Has(CustomOIDCConfigurationIDKey)).To(BeTrue())
	g.Expect(leftLens.Has(CustomOIDCConfigurationIDKey)).To(BeTrue())
	g.Expect(leftLens.Has(LensKey)).To(BeFalse())
}
