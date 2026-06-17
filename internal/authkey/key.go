package authkey

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const keyPrefix = "distr-"

type Key [16]byte

var ErrInvalidAccessKey = errors.New("invalid access key")

func Parse(encoded string) (Key, error) {
	if !strings.HasPrefix(encoded, keyPrefix) {
		return Key{}, ErrInvalidAccessKey
	} else if decoded, err := hex.DecodeString(strings.TrimPrefix(encoded, keyPrefix)); err != nil {
		return Key{}, fmt.Errorf("%w: %w", ErrInvalidAccessKey, err)
	} else if len(decoded) != 16 {
		return Key{}, ErrInvalidAccessKey
	} else {
		return Key(decoded), nil
	}
}

func NewKey() (key Key, err error) {
	_, err = rand.Read(key[:])
	return key, err
}

func (key Key) String() string {
	return keyPrefix + hex.EncodeToString(key[:3]) + "___REDACTED___"
}

func (key Key) Serialize() string { return keyPrefix + hex.EncodeToString(key[:]) }

func (key Key) MarshalJSON() ([]byte, error) { return json.Marshal(key.Serialize()) }

func (key *Key) Scan(src any) error {
	switch v := src.(type) {
	case []byte:
		if len(v) == 16 {
			*key = Key(slices.Clone(v))
			return nil
		}
	}
	return errors.New("cannot scan into Key")
}
