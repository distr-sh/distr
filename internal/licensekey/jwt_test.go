package licensekey

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	. "github.com/onsi/gomega"
)

// Generated with: openssl genpkey -algorithm ed25519 | base64 -w0
const testPrivateKeyPEM = `
-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEID0kZeYRX/KmYFMZNfHly8KODNz8dDKQfPn33W0gkori
-----END PRIVATE KEY-----
`

func testKey(t *testing.T) jwk.Key {
	t.Helper()
	key, err := jwk.ParseKey([]byte(testPrivateKeyPEM), jwk.WithPEM(true))
	NewWithT(t).Expect(err).ToNot(HaveOccurred())
	return key
}

func TestGenerateToken(t *testing.T) {
	g := NewWithT(t)
	key := testKey(t)
	now := time.Now().Truncate(time.Second)
	licenseKey := &types.LicenseKey{
		ID:        uuid.New(),
		CreatedAt: now,
		NotBefore: now,
		ExpiresAt: now.Add(24 * time.Hour),
		Payload:   json.RawMessage(`{"plan":"enterprise"}`),
	}

	token, err := generateToken(key, licenseKey, "test-issuer")
	g.Expect(err).ToNot(HaveOccurred())

	pubKey, err := key.PublicKey()
	g.Expect(err).ToNot(HaveOccurred())

	parsed, err := jwt.Parse([]byte(token), jwt.WithKey(jwa.EdDSA(), pubKey))
	g.Expect(err).ToNot(HaveOccurred())

	subject, ok := parsed.Subject()
	g.Expect(ok).To(BeTrue())
	g.Expect(subject).To(Equal(licenseKey.ID.String()))

	issuer, ok := parsed.Issuer()
	g.Expect(ok).To(BeTrue())
	g.Expect(issuer).To(Equal("test-issuer"))

	var plan string
	err = parsed.Get("plan", &plan)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(plan).To(Equal("enterprise"))
}

func TestGenerateToken_ReservedClaimsStripped(t *testing.T) {
	g := NewWithT(t)
	key := testKey(t)
	now := time.Now().Truncate(time.Second)
	licenseKey := &types.LicenseKey{
		ID:        uuid.New(),
		CreatedAt: now,
		NotBefore: now,
		ExpiresAt: now.Add(24 * time.Hour),
		Payload:   json.RawMessage(`{"exp":99999,"plan":"pro"}`),
	}

	token, err := generateToken(key, licenseKey, "test-issuer")
	g.Expect(err).ToNot(HaveOccurred())

	pubKey, _ := key.PublicKey()
	parsed, err := jwt.Parse([]byte(token), jwt.WithKey(jwa.EdDSA(), pubKey))
	g.Expect(err).ToNot(HaveOccurred())

	// exp must be the one from licenseKey.ExpiresAt, not the payload override
	exp, ok := parsed.Expiration()
	g.Expect(ok).To(BeTrue())
	g.Expect(exp.UTC()).To(Equal(licenseKey.ExpiresAt.UTC()))
}

func TestValidatePayload(t *testing.T) {
	g := NewWithT(t)
	g.Expect(ValidatePayload(json.RawMessage(`{"foo":"bar"}`))).To(Succeed())
	g.Expect(ValidatePayload(json.RawMessage(`{"exp":12345}`))).To(HaveOccurred())
	g.Expect(ValidatePayload(json.RawMessage(`not-json`))).To(HaveOccurred())
}
