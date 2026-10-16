package authkey

import (
	"crypto/rand"
	"crypto/sha256"
)

type Secret [32]byte

// SecureToken carries an additional secret.
type SecureToken struct {
	key    Key
	secret Secret
}

func NewToken() (SecureToken, error) {
	key, err := NewKey()
	if err != nil {
		return SecureToken{}, err
	}
	secret, err := NewSecret()
	if err != nil {
		return SecureToken{}, err
	}
	return NewSecureToken(key, secret), nil
}

func NewSecureToken(key Key, secret Secret) SecureToken {
	return SecureToken{key: key, secret: secret}
}

func (token SecureToken) Key() Key { return token.key }

func (token SecureToken) Secret() Secret { return token.secret }

func (token SecureToken) ID() string { return token.key.ID() }

func (token SecureToken) Serialize() string {
	body := base62Encode(token.key[:], keyEncodedLen) +
		secretSeparator + base62Encode(token.secret[:], secretEncodedLen)
	return keyPrefix + body + checksum(body)
}

func (token SecureToken) String() string { return token.key.String() }

func NewSecret() (secret Secret, err error) {
	_, err = rand.Read(secret[:])
	return secret, err
}

func parseSecret(encoded string) (Secret, error) {
	if len(encoded) != secretEncodedLen {
		return Secret{}, ErrInvalidAccessKey
	}
	decoded, err := base62Decode(encoded, len(Secret{}))
	if err != nil {
		return Secret{}, err
	}
	return Secret(decoded), nil
}

func (secret Secret) String() string { return "___REDACTED___" }

// Hash derives the value stored in the database, and the value a stored one is looked up by. Unlike
// a password, a secret is 256 bits of CSPRNG output, so there is no dictionary to search and nothing
// a salt could keep apart: one SHA-256 keeps verification affordable on every single request, and
// leaves the comparison to the database.
func (secret Secret) Hash() []byte {
	hash := sha256.Sum256(secret[:])
	return hash[:]
}
