package usagelicense

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var registeredClaims = map[string]struct{}{
	"exp": {}, "nbf": {}, "iss": {}, "sub": {}, "aud": {},
	"iat": {}, "jti": {},
}

func GenerateToken(license *types.UsageLicense, issuer string) (string, error) {
	var customClaims map[string]any
	if err := json.Unmarshal(license.Payload, &customClaims); err != nil {
		return "", fmt.Errorf("invalid payload JSON: %w", err)
	}
	for k := range registeredClaims {
		delete(customClaims, k)
	}

	builder := jwt.NewBuilder().
		Issuer(issuer).
		Subject(license.ID.String()).
		Audience([]string{"usage-license"}).
		IssuedAt(time.Now()).
		JwtID(uuid.New().String()).
		NotBefore(license.NotBefore).
		Expiration(license.ExpiresAt)

	for k, v := range customClaims {
		builder = builder.Claim(k, v)
	}

	token, err := builder.Build()
	if err != nil {
		return "", fmt.Errorf("could not build JWT: %w", err)
	}

	privateKey := env.UsageLicensePrivateKey()
	if privateKey == nil {
		unsigned, err := jwt.Sign(token, jwt.WithInsecureNoSignature())
		if err != nil {
			return "", fmt.Errorf("could not serialize unsigned JWT: %w", err)
		}
		return string(unsigned), nil
	}

	key, err := jwk.FromRaw(privateKey)
	if err != nil {
		return "", fmt.Errorf("could not create JWK from private key: %w", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.EdDSA, key))
	if err != nil {
		return "", fmt.Errorf("could not sign JWT: %w", err)
	}

	return string(signed), nil
}

func ValidatePayload(payload json.RawMessage) error {
	var raw any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return fmt.Errorf("invalid JSON payload: %w", err)
	}

	obj, ok := raw.(map[string]any)
	if !ok {
		return errors.New("payload must be a JSON object")
	}

	for k := range obj {
		if _, reserved := registeredClaims[k]; reserved {
			return fmt.Errorf("payload must not contain registered JWT claim %q", k)
		}
	}
	return nil
}
