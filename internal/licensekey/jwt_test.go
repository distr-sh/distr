package licensekey

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Generated with: openssl genpkey -algorithm ed25519 | base64 -w0
const testPrivateKeyPEMB64 = "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1DNENBUUF3QlFZREsy" +
	"VndCQ0lFSUQwa1plWVJYL0ttWUZNWk5mSGx5OEtPRE56OGJES1FmUG4z" +
	"M1cwZ2tvcmkKLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo="

func testKey(t *testing.T) jwk.Key {
	t.Helper()
	pemBytes, err := base64.StdEncoding.DecodeString(testPrivateKeyPEMB64)
	if err != nil {
		t.Fatal(err)
	}
	key, err := jwk.ParseKey(pemBytes, jwk.WithPEM(true))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestGenerateToken(t *testing.T) {
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
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}

	pubKey, err := key.PublicKey()
	if err != nil {
		t.Fatalf("key.PublicKey: %v", err)
	}
	parsed, err := jwt.Parse([]byte(token), jwt.WithKey(jwa.EdDSA, pubKey))
	if err != nil {
		t.Fatalf("jwt.Parse: %v", err)
	}

	if parsed.Subject() != licenseKey.ID.String() {
		t.Errorf("subject = %q, want %q", parsed.Subject(), licenseKey.ID.String())
	}
	if parsed.Issuer() != "test-issuer" {
		t.Errorf("issuer = %q, want %q", parsed.Issuer(), "test-issuer")
	}
	if v, ok := parsed.Get("plan"); !ok || v != "enterprise" {
		t.Errorf("claim plan = %v, want %q", v, "enterprise")
	}
}

func TestGenerateToken_ReservedClaimsStripped(t *testing.T) {
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
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}

	pubKey, _ := key.PublicKey()
	parsed, err := jwt.Parse([]byte(token), jwt.WithKey(jwa.EdDSA, pubKey))
	if err != nil {
		t.Fatalf("jwt.Parse: %v", err)
	}

	// exp must be the one from licenseKey.ExpiresAt, not the payload override
	if !parsed.Expiration().Equal(licenseKey.ExpiresAt) {
		t.Errorf("expiration = %v, want %v", parsed.Expiration(), licenseKey.ExpiresAt)
	}
}

func TestValidatePayload(t *testing.T) {
	if err := ValidatePayload(json.RawMessage(`{"foo":"bar"}`)); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := ValidatePayload(json.RawMessage(`{"exp":12345}`)); err == nil {
		t.Error("expected error for reserved claim 'exp'")
	}
	if err := ValidatePayload(json.RawMessage(`not-json`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
