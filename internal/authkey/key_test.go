package authkey

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestKeyID asserts that what a token's owner is shown of it is a prefix of the token itself, so
// that it identifies one, and that it stops well before the whole key, so that a token which is
// nothing but its key cannot be reassembled from what identifies it.
func TestKeyID(t *testing.T) {
	g := NewWithT(t)

	token, err := NewToken()
	g.Expect(err).ToNot(HaveOccurred())

	legacy := LegacyToken{key: token.key}

	g.Expect(token.Serialize()).To(HavePrefix(token.ID()))
	g.Expect(legacy.Serialize()).To(HavePrefix(legacy.ID()))

	g.Expect(token.ID()).To(HaveLen(len(keyPrefix) + keyEncodedLen/2))
	g.Expect(legacy.ID()).To(HaveLen(len(keyPrefix) + legacyKeyEncodedLen/2))
}
