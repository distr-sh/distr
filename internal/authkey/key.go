package authkey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"math/big"
	"slices"
	"strings"
)

const (
	keyPrefix       = "distr-"
	secretSeparator = "_"
	// base62 is dense enough to keep a token short, and unlike base64 its alphabet contains neither
	// the separator between key and secret nor a character that has to be escaped in a URL.
	base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	// The encoded lengths of a key, a secret and the checksum. Each covers its value with room to
	// spare, and is fixed so that a token has one length rather than a shorter one for a value with
	// leading zero bytes.
	keyEncodedLen      = 22
	secretEncodedLen   = 43
	checksumEncodedLen = 6
	// legacyKeyEncodedLen is the length of the hex encoding that predates the secret. Hex is a subset
	// of the base62 alphabet, so the two encodings of a key are told apart by their length.
	legacyKeyEncodedLen = 2 * len(Key{})
	// The lengths of what a token's owner is shown of its key. Half of the encoding tells their
	// tokens apart, and leaves a token that predates secrets, which is nothing but its key, with
	// too little of it disclosed to authenticate.
	keyIDEncodedLen       = keyEncodedLen / 2
	legacyKeyIDEncodedLen = legacyKeyEncodedLen / 2
)

type Key [16]byte

type Secret [32]byte

type Token interface {
	Key() Key
	// ID is the part of the token that identifies the token to its owner.
	ID() string
	Serialize() string
	String() string
}

// SecureToken carries an additional secret.
type SecureToken struct {
	key    Key
	secret Secret
}

// LegacyToken was issued before secrets existed and only authenticates a row that has none.
//
// Deprecated: Only tokens issued before secrets existed are parsed into it. Issue a [SecureToken].
type LegacyToken struct {
	key Key
}

var ErrInvalidAccessKey = errors.New("invalid access key")

func Parse(encoded string) (Token, error) {
	body, ok := strings.CutPrefix(encoded, keyPrefix)
	if !ok {
		return nil, ErrInvalidAccessKey
	}

	keyPart, rest, hasSecret := strings.Cut(body, secretSeparator)
	if !hasSecret {
		key, err := parseLegacyKey(keyPart)
		if err != nil {
			return nil, err
		}
		return LegacyToken{key: key}, nil
	}

	if len(rest) != secretEncodedLen+checksumEncodedLen {
		return nil, ErrInvalidAccessKey
	}
	secretPart, sum := rest[:secretEncodedLen], rest[secretEncodedLen:]
	if sum != checksum(keyPart+secretSeparator+secretPart) {
		return nil, ErrInvalidAccessKey
	}

	key, err := parseKey(keyPart)
	if err != nil {
		return nil, err
	}
	secret, err := parseSecret(secretPart)
	if err != nil {
		return nil, err
	}
	return SecureToken{key: key, secret: secret}, nil
}

// checksum is what lets a token that was mistyped, truncated or line wrapped be rejected without a
// database lookup, and lets a secret scanner recognize one of ours without asking us. It is not a
// security measure: the algorithm is public, so anyone can produce a token that passes it.
func checksum(body string) string {
	sum := crc32.ChecksumIEEE([]byte(body))
	return base62Encode(binary.BigEndian.AppendUint32(nil, sum), checksumEncodedLen)
}

// base62Encode renders value as exactly width characters, left-padded with the first character of
// the alphabet. The widths of this package all cover their value, which [TestBase62Widths] asserts.
func base62Encode(value []byte, width int) string {
	number := new(big.Int).SetBytes(value)
	base := big.NewInt(int64(len(base62Alphabet)))
	digit := new(big.Int)
	encoded := make([]byte, width)
	for i := width - 1; i >= 0; i-- {
		number.QuoRem(number, base, digit)
		encoded[i] = base62Alphabet[digit.Int64()]
	}
	return string(encoded)
}

// base62Decode is the inverse of [base62Encode] for a value of exactly size bytes, rejecting anything
// outside the alphabet as well as a number too large to be one.
func base62Decode(encoded string, size int) ([]byte, error) {
	number := new(big.Int)
	base := big.NewInt(int64(len(base62Alphabet)))
	for i := range len(encoded) {
		digit := strings.IndexByte(base62Alphabet, encoded[i])
		if digit < 0 {
			return nil, ErrInvalidAccessKey
		}
		number.Add(number.Mul(number, base), big.NewInt(int64(digit)))
	}
	if len(number.Bytes()) > size {
		return nil, ErrInvalidAccessKey
	}
	return number.FillBytes(make([]byte, size)), nil
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

func (token LegacyToken) Key() Key { return token.key }

func (token LegacyToken) ID() string { return token.key.LegacyID() }

func (token LegacyToken) Serialize() string { return keyPrefix + hex.EncodeToString(token.key[:]) }

func (token LegacyToken) String() string { return token.key.String() }

func NewKey() (key Key, err error) {
	_, err = rand.Read(key[:])
	return key, err
}

func parseKey(encoded string) (Key, error) {
	if len(encoded) != keyEncodedLen {
		return Key{}, ErrInvalidAccessKey
	}
	decoded, err := base62Decode(encoded, len(Key{}))
	if err != nil {
		return Key{}, err
	}
	return Key(decoded), nil
}

func parseLegacyKey(encoded string) (Key, error) {
	if len(encoded) != legacyKeyEncodedLen {
		return Key{}, ErrInvalidAccessKey
	}
	if decoded, err := hex.DecodeString(encoded); err != nil {
		return Key{}, fmt.Errorf("%w: %w", ErrInvalidAccessKey, err)
	} else {
		return Key(decoded), nil
	}
}

func (key Key) String() string {
	return keyPrefix + base62Encode(key[:], keyEncodedLen)[:5] + "___REDACTED___"
}

// ID renders the beginning of the key as it appears at the beginning of a token, which is how a
// token is identified to the user it belongs to.
func (key Key) ID() string { return keyPrefix + base62Encode(key[:], keyEncodedLen)[:keyIDEncodedLen] }

// LegacyID is [Key.ID] in the hex encoding that predates secrets, which is the only form in which
// the owner of such a token has ever seen it.
func (key Key) LegacyID() string {
	return keyPrefix + hex.EncodeToString(key[:])[:legacyKeyIDEncodedLen]
}

func (key *Key) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		if len(v) == len(Key{}) {
			*key = Key(slices.Clone(v))
			return nil
		}
	}
	return errors.New("cannot scan into Key")
}

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
