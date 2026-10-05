package authkey

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestSecretHash(t *testing.T) {
	g := NewWithT(t)

	secret, err := NewSecret()
	g.Expect(err).ToNot(HaveOccurred())
	other, err := NewSecret()
	g.Expect(err).ToNot(HaveOccurred())

	// The stored value is what a lookup compares against, so it has to be derived from the secret
	// alone, and never be the secret itself.
	g.Expect(secret.Hash()).To(Equal(secret.Hash()))
	g.Expect(secret.Hash()).ToNot(Equal(secret[:]))
	g.Expect(secret.Hash()).ToNot(Equal(other.Hash()))
}
