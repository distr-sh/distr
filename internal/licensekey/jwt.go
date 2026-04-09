package licensekey

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

var registeredClaims = map[string]struct{}{
	jwt.ExpirationKey: {}, jwt.NotBeforeKey: {}, jwt.IssuerKey: {},
	jwt.SubjectKey: {}, jwt.AudienceKey: {}, jwt.IssuedAtKey: {},
}

var ErrNoSigningKey = errors.New("no license key signing key configured")

var signingKey = sync.OnceValues(func() (jwk.Key, error) {
	pemBytes := env.LicenseKeyPrivateKey()
	if pemBytes == nil {
		return nil, ErrNoSigningKey
	}
	return jwk.ParseKey(pemBytes, jwk.WithPEM(true))
})

func PublicKey() (jwk.Key, error) {
	if k, err := signingKey(); err != nil {
		return nil, err
	} else {
		return k.PublicKey()
	}
}

func IsSigningKeyConfigured() bool {
	return env.LicenseKeyPrivateKey() != nil
}

type LicenseKeyData struct {
	LicenseKeyID uuid.UUID
	IssuedAt     time.Time
	ExpiresAt    time.Time
	NotBefore    time.Time
	Payload      json.RawMessage
}

func FromLicenseKey(lk types.LicenseKey) LicenseKeyData {
	var issuedAt, expiresAt, notBefore time.Time
	if lk.LastRevisedAt != nil {
		issuedAt = *lk.LastRevisedAt
	}
	if lk.ExpiresAt != nil {
		expiresAt = *lk.ExpiresAt
	}
	if lk.NotBefore != nil {
		notBefore = *lk.NotBefore
	}
	return LicenseKeyData{
		LicenseKeyID: lk.ID,
		IssuedAt:     issuedAt,
		ExpiresAt:    expiresAt,
		NotBefore:    notBefore,
		Payload:      lk.Payload,
	}
}

func FromLicenseKeyAndRevision(lk types.LicenseKey, r types.LicenseKeyRevision) LicenseKeyData {
	return LicenseKeyData{
		LicenseKeyID: lk.ID,
		IssuedAt:     r.CreatedAt,
		ExpiresAt:    r.ExpiresAt,
		NotBefore:    r.NotBefore,
		Payload:      r.Payload,
	}
}

func GenerateToken(src LicenseKeyData, issuer string) (string, error) {
	key, err := signingKey()
	if err != nil {
		return "", err
	}
	return generateToken(key, src, issuer)
}

func generateToken(key jwk.Key, src LicenseKeyData, issuer string) (string, error) {
	var customClaims map[string]any
	if err := json.Unmarshal(src.Payload, &customClaims); err != nil {
		return "", fmt.Errorf("invalid payload JSON: %w", err)
	}
	for k := range registeredClaims {
		delete(customClaims, k)
	}

	builder := jwt.NewBuilder().
		Issuer(issuer).
		Subject(src.LicenseKeyID.String()).
		Audience([]string{"license-key"}).
		IssuedAt(src.IssuedAt).
		NotBefore(src.NotBefore).
		Expiration(src.ExpiresAt)

	for k, v := range customClaims {
		builder = builder.Claim(k, v)
	}

	token, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("could not build JWT: %w", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.EdDSA(), key))
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
