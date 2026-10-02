package authkey

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"slices"
)

type Key [16]byte

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
