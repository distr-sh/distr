package license

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/distr-sh/distr/internal/limit"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	. "github.com/onsi/gomega"
)

// Fixed 32-byte seed for deterministic Ed25519 key generation in tests.
var testSeed = [32]byte{
	0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
	0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
}

func testKeyPair(t *testing.T) (jwk.Key, ed25519.PrivateKey) {
	t.Helper()
	privKey := ed25519.NewKeyFromSeed(testSeed[:])
	pubKey := privKey.Public().(ed25519.PublicKey)

	der, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		t.Fatal(err)
	}
	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	jwkPubKey, err := jwk.ParseKey(pemBlock, jwk.WithPEM(true))
	if err != nil {
		t.Fatal(err)
	}
	return jwkPubKey, privKey
}

func signToken(t *testing.T, privKey ed25519.PrivateKey, claims map[string]any) string {
	t.Helper()
	b := jwt.NewBuilder().Expiration(time.Now().Add(time.Hour))
	for k, v := range claims {
		b = b.Claim(k, v)
	}
	tok, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	privJWK, err := jwk.FromRaw(privKey)
	if err != nil {
		t.Fatal(err)
	}

	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.EdDSA, privJWK))
	if err != nil {
		t.Fatal(err)
	}
	return string(signed)
}

func pubKeyFunc(key jwk.Key) func() (jwk.Key, error) {
	return func() (jwk.Key, error) { return key, nil }
}

func nilPubKeyFunc() (jwk.Key, error) { return nil, nil }

func errPubKeyFunc() (jwk.Key, error) { return nil, errors.New("key load error") }

func TestParseAndValidate_NoPubKey_ReturnsDefault(t *testing.T) {
	g := NewWithT(t)
	got, err := parseAndValidate(nilPubKeyFunc, "")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(*got).To(Equal(defaultLicenseData))
}

func TestParseAndValidate_PubKeyLoadError(t *testing.T) {
	g := NewWithT(t)
	_, err := parseAndValidate(errPubKeyFunc, "")
	g.Expect(err).To(HaveOccurred())
}

func TestParseAndValidate_PubKeySet_EmptyLicenseKey(t *testing.T) {
	g := NewWithT(t)
	pub, _ := testKeyPair(t)
	_, err := parseAndValidate(pubKeyFunc(pub), "")
	g.Expect(err).To(HaveOccurred())
}

func TestParseAndValidate_InvalidToken(t *testing.T) {
	g := NewWithT(t)
	pub, _ := testKeyPair(t)
	_, err := parseAndValidate(pubKeyFunc(pub), "not.a.jwt")
	g.Expect(err).To(HaveOccurred())
}

func TestParseAndValidate_WrongKey(t *testing.T) {
	g := NewWithT(t)
	pub, _ := testKeyPair(t)

	otherSeed := [32]byte{0xff, 0xfe, 0xfd}
	otherPriv := ed25519.NewKeyFromSeed(otherSeed[:])
	token := signToken(t, otherPriv, map[string]any{"enf": true})

	_, err := parseAndValidate(pubKeyFunc(pub), token)
	g.Expect(err).To(HaveOccurred())
}

func TestParseAndValidate_AllFields(t *testing.T) {
	g := NewWithT(t)
	pub, priv := testKeyPair(t)
	token := signToken(t, priv, map[string]any{
		"enf": true,
		"mo":  10,
		"mou": 20,
		"moc": 30,
		"mcu": 40,
		"mcd": 50,
		"mlr": 60,
	})

	got, err := parseAndValidate(pubKeyFunc(pub), token)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(*got).To(Equal(LicenseData{
		EnforceLimitsOnStartup:                      true,
		MaxOrganizations:                            limit.New(10),
		MaxUsersPerOrganization:                     limit.New(20),
		MaxCustomersPerOrganization:                 limit.New(30),
		MaxUsersPerCustomerOrganization:             limit.New(40),
		MaxDeploymentTargetsPerCustomerOrganization: limit.New(50),
		MaxLogExportRows:                            limit.New(60),
	}))
}

func TestParseAndValidate_PartialClaims_DefaultsForUnspecifiedFields(t *testing.T) {
	g := NewWithT(t)
	pub, priv := testKeyPair(t)
	token := signToken(t, priv, map[string]any{
		"enf": false,
		"mo":  5,
	})

	got, err := parseAndValidate(pubKeyFunc(pub), token)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(*got).To(Equal(LicenseData{
		EnforceLimitsOnStartup:                      false,
		MaxOrganizations:                            limit.New(5),
		MaxUsersPerOrganization:                     defaultLicenseData.MaxUsersPerOrganization,
		MaxCustomersPerOrganization:                 defaultLicenseData.MaxCustomersPerOrganization,
		MaxUsersPerCustomerOrganization:             defaultLicenseData.MaxUsersPerCustomerOrganization,
		MaxDeploymentTargetsPerCustomerOrganization: defaultLicenseData.MaxDeploymentTargetsPerCustomerOrganization,
		MaxLogExportRows:                            defaultLicenseData.MaxLogExportRows,
	}))
}

func TestParseAndValidate_ZeroLimits(t *testing.T) {
	g := NewWithT(t)
	pub, priv := testKeyPair(t)
	token := signToken(t, priv, map[string]any{
		"enf": false,
		"mo":  0,
		"mou": 0,
		"moc": 0,
		"mcu": 0,
		"mcd": 0,
		"mlr": 0,
	})

	got, err := parseAndValidate(pubKeyFunc(pub), token)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(*got).To(Equal(LicenseData{
		EnforceLimitsOnStartup:                      false,
		MaxOrganizations:                            limit.New(0),
		MaxUsersPerOrganization:                     limit.New(0),
		MaxCustomersPerOrganization:                 limit.New(0),
		MaxUsersPerCustomerOrganization:             limit.New(0),
		MaxDeploymentTargetsPerCustomerOrganization: limit.New(0),
		MaxLogExportRows:                            limit.New(0),
	}))
}
