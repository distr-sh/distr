package apitest_test

import (
	"net/http"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	. "github.com/onsi/gomega"
)

func TestSecretValueIsOnlyStoredEncrypted(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	secret := vendor.CreateSecret("TOKEN", "first-plaintext", nil)
	g := NewWithT(t)

	expectEncrypted := func(plaintext string) {
		t.Helper()
		g.Expect(apitest.QueryValue[bool](h,
			`SELECT value IS NULL AND value_enc IS NOT NULL AND position(convert_to($2, 'UTF8') IN value_enc) = 0
			FROM Secret WHERE id = $1`,
			secret, plaintext,
		)).To(BeTrue())
	}
	expectEncrypted("first-plaintext")

	vendor.Put(apitest.Path(apitest.Secrets, secret), api.UpdateSecretRequest{Value: "second-plaintext"}).
		ExpectStatus(http.StatusOK)
	expectEncrypted("second-plaintext")

	g.Expect(string(vendor.Get(apitest.Secrets).ExpectStatus(http.StatusOK).Body)).
		NotTo(ContainSubstring("plaintext"))
}
