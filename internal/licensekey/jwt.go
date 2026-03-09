package licensekey

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var registeredClaims = map[string]struct{}{
	jwt.ExpirationKey: {}, jwt.NotBeforeKey: {}, jwt.IssuerKey: {},
	jwt.SubjectKey: {}, jwt.AudienceKey: {}, jwt.IssuedAtKey: {},
}

var signingKey = sync.OnceValues(func() (jwk.Key, error) {
	pemBytes := env.LicenseKeyPrivateKey()
	if pemBytes == nil {
		return nil, errors.New("no license key signing key configured")
	}
	return jwk.ParseKey(pemBytes, jwk.WithPEM(true))
})

func IsSigningKeyConfigured() bool {
	return env.LicenseKeyPrivateKey() != nil
}

func GenerateToken(licenseKey *types.LicenseKey, issuer string) (string, error) {
	key, err := signingKey()
	if err != nil {
		return "", err
	}
	return generateToken(key, licenseKey, issuer)
}

func generateToken(key jwk.Key, licenseKey *types.LicenseKey, issuer string) (string, error) {
	var customClaims map[string]any
	if err := json.Unmarshal(licenseKey.Payload, &customClaims); err != nil {
		return "", fmt.Errorf("invalid payload JSON: %w", err)
	}
	for k := range registeredClaims {
		delete(customClaims, k)
	}

	builder := jwt.NewBuilder().
		Issuer(issuer).
		Subject(licenseKey.ID.String()).
		Audience([]string{"license-key"}).
		IssuedAt(licenseKey.CreatedAt).
		NotBefore(licenseKey.NotBefore).
		Expiration(licenseKey.ExpiresAt)

	for k, v := range customClaims {
		builder = builder.Claim(k, v)
	}

	token, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("could not build JWT: %w", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.EdDSA, key))
	if err != nil {
		return "", fmt.Errorf("could not sign JWT: %w", err)
	}

	return string(signed), nil
}

func ValidatePayload(payload json.RawMessage) error {
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}

	for k := range raw {
		if _, reserved := registeredClaims[k]; reserved {
			return fmt.Errorf("payload must not contain registered JWT claim %q", k)
		}
	}
	return nil
}
