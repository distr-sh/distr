package authkey

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func TestParse(t *testing.T) {
	g := NewWithT(t)

	token, err := NewToken()
	g.Expect(err).ToNot(HaveOccurred())

	serialized := token.Serialize()
	g.Expect(serialized).To(HaveLen(
		len(keyPrefix) + keyEncodedLen + len(secretSeparator) + secretEncodedLen + checksumEncodedLen))

	parsed, err := Parse(serialized)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(parsed.Key).To(Equal(token.Key))
	g.Expect(parsed.Secret).To(HaveValue(Equal(*token.Secret)))

	// Tokens issued before secrets existed carry the key only, hex encoded.
	legacy, err := Parse(Token{Key: token.Key}.Serialize())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(legacy.Key).To(Equal(token.Key))
	g.Expect(legacy.Secret).To(BeNil())

	body := strings.TrimPrefix(serialized, keyPrefix)
	key, secret := body[:keyEncodedLen], body[keyEncodedLen+1:keyEncodedLen+1+secretEncodedLen]
	valid := key + "_" + secret
	for name, encoded := range map[string]string{
		"no prefix":            valid + checksum(valid),
		"nothing but prefix":   keyPrefix,
		"key too short":        keyPrefix + key[1:] + "_" + secret + checksum(key[1:]+"_"+secret),
		"key outside alphabet": keyPrefix + "+" + key[1:] + "_" + secret + checksum("+"+key[1:]+"_"+secret),
		"secret too short":     keyPrefix + key + "_" + secret[1:] + checksum(key+"_"+secret[1:]),
		"empty secret":         keyPrefix + key + "_",
		"missing checksum":     keyPrefix + valid,
		"wrong checksum":       keyPrefix + valid + checksum(valid+"x"),
		"tampered key":         keyPrefix + "0" + key[1:] + "_" + secret + checksum(valid),
		"second separator":     keyPrefix + valid + "_" + checksum(valid),
		"legacy key too short": keyPrefix + hex.EncodeToString(token.Key[1:]),
		"legacy key not hex":   keyPrefix + "zz" + hex.EncodeToString(token.Key[:])[2:],
	} {
		_, err := Parse(encoded)
		g.Expect(err).To(MatchError(ErrInvalidAccessKey), name)
	}
}

// TestBase62Widths asserts that the encoded lengths cover the largest value of each size, since
// base62Encode silently drops what does not fit into its width.
func TestBase62Widths(t *testing.T) {
	g := NewWithT(t)

	for name, format := range map[string]struct{ size, width int }{
		"key":      {len(Key{}), keyEncodedLen},
		"secret":   {len(Secret{}), secretEncodedLen},
		"checksum": {4, checksumEncodedLen},
	} {
		largest := bytes.Repeat([]byte{0xff}, format.size)
		decoded, err := base62Decode(base62Encode(largest, format.width), format.size)
		g.Expect(err).ToNot(HaveOccurred(), name)
		g.Expect(decoded).To(Equal(largest), name)

		zero := make([]byte, format.size)
		g.Expect(base62Encode(zero, format.width)).To(HaveLen(format.width), name)
		g.Expect(base62Decode(base62Encode(zero, format.width), format.size)).To(Equal(zero), name)
	}
}

func TestVerifySecret(t *testing.T) {
	g := NewWithT(t)

	secret, err := NewSecret()
	g.Expect(err).ToNot(HaveOccurred())
	other, err := NewSecret()
	g.Expect(err).ToNot(HaveOccurred())
	salt, err := NewSalt()
	g.Expect(err).ToNot(HaveOccurred())
	otherSalt, err := NewSalt()
	g.Expect(err).ToNot(HaveOccurred())

	hash := secret.Hash(salt)
	g.Expect(hash).ToNot(Equal(secret[:]))
	g.Expect(VerifySecret(salt, hash, secret)).To(BeTrue())
	g.Expect(VerifySecret(salt, hash, other)).To(BeFalse())
	g.Expect(VerifySecret(otherSalt, hash, secret)).To(BeFalse())
}
