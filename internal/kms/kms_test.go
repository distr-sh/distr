package kms

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

type fakeProvider struct {
	plaintext  string
	ciphertext []byte
	setting    string
	closed     bool
}

func (p *fakeProvider) Decrypt(_ context.Context, ciphertext []byte, setting string) ([]byte, error) {
	p.ciphertext = ciphertext
	p.setting = setting
	if p.plaintext == "" {
		return nil, errors.New("no such ciphertext")
	}
	return []byte(p.plaintext), nil
}

func (p *fakeProvider) Close() error {
	p.closed = true
	return nil
}

// TestEncryptionContext pins the strings an operator reproduces with their cloud CLI to wrap a
// value: changing either locks every instance out of what it has already wrapped.
func TestEncryptionContext(t *testing.T) {
	g := NewWithT(t)
	g.Expect(encryptionContext("DATABASE_ENCRYPTION_KEY")).
		To(Equal(map[string]string{"distr": "DATABASE_ENCRYPTION_KEY"}))
	g.Expect(string(additionalAuthenticatedData("JWT_SECRET"))).To(Equal("distr=JWT_SECRET"))
}

func TestResolve(t *testing.T) {
	t.Run("returns a value that is not wrapped unchanged", func(t *testing.T) {
		g := NewWithT(t)
		provider := &fakeProvider{plaintext: "unwrapped"}
		resolver := Resolver{provider: provider}
		g.Expect(resolver.Resolve(t.Context(), "DATABASE_ENCRYPTION_KEY", "0:c2VjcmV0")).
			To(Equal("0:c2VjcmV0"))
		g.Expect(provider.setting).To(BeEmpty())
	})

	t.Run("unwraps a wrapped value, bound to the setting it is read for", func(t *testing.T) {
		g := NewWithT(t)
		provider := &fakeProvider{plaintext: "0:the key"}
		resolver := Resolver{provider: provider}
		ciphertext := []byte{0x01, 0x02, 0x03}
		value := Prefix + base64.StdEncoding.EncodeToString(ciphertext)
		g.Expect(resolver.Resolve(t.Context(), "DATABASE_ENCRYPTION_KEY", value)).To(Equal("0:the key"))
		g.Expect(provider.ciphertext).To(Equal(ciphertext))
		g.Expect(provider.setting).To(Equal("DATABASE_ENCRYPTION_KEY"))
	})

	t.Run("rejects a wrapped value when no service is configured", func(t *testing.T) {
		g := NewWithT(t)
		resolver, err := New(t.Context(), Config{})
		g.Expect(err).NotTo(HaveOccurred())
		_, err = resolver.Resolve(t.Context(), "JWT_SECRET", Prefix+"c2VjcmV0")
		g.Expect(err).To(MatchError(ContainSubstring("KMS_AWS_KEY_ID, KMS_GCP_KEY_NAME")))
	})

	t.Run("rejects a wrapped value that is not base64", func(t *testing.T) {
		g := NewWithT(t)
		resolver := Resolver{provider: &fakeProvider{plaintext: "0:the key"}}
		_, err := resolver.Resolve(t.Context(), "JWT_SECRET", Prefix+"not base64!")
		g.Expect(err).To(MatchError(ContainSubstring("JWT_SECRET is not valid base64")))
	})
}

func TestNewRejectsTwoServices(t *testing.T) {
	g := NewWithT(t)
	_, err := New(t.Context(), Config{
		AWS: AWSConfig{KeyID: "alias/distr"},
		GCP: GCPConfig{KeyName: "projects/p/locations/l/keyRings/r/cryptoKeys/k"},
	})
	g.Expect(err).To(MatchError(ContainSubstring("KMS_AWS_KEY_ID and KMS_GCP_KEY_NAME")))
}
