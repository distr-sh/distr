package authkey

import (
	"encoding/hex"
	"fmt"
)

// LegacyToken was issued before secrets existed and only authenticates a row that has none.
//
// Deprecated: Only tokens issued before secrets existed are parsed into it. Issue a [SecureToken].
type LegacyToken struct {
	key Key
}

func (token LegacyToken) Key() Key { return token.key }

func (token LegacyToken) ID() string { return token.key.LegacyID() }

func (token LegacyToken) Serialize() string { return keyPrefix + hex.EncodeToString(token.key[:]) }

func (token LegacyToken) String() string { return token.key.String() }

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
