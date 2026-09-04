package dbcrypto

import (
	"bytes"
	"compress/flate"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/glasskube/pkg/crypto"
)

// The stored value of an encrypted column is FormatVersion, the ID of the key it was sealed with, a
// flags byte, and then the ciphertext of [crypto.Encryptor]. A value that a query read from the
// plaintext column instead is prefixed with plaintextMarker by [TextValue] and [BytesValue], which is
// how a read tells the two apart without a second column in the result.
const (
	plaintextMarker byte = 0
	// FormatVersion is the first byte of every encrypted value, and the key id is the second. A
	// maintenance query relies on this to find the values of a retired key without decrypting them.
	FormatVersion byte = 1
)

const headerLen = 3

const flagCompressed byte = 1 << 0

// Each configured key is expanded into two independent subkeys, so the AES key and the HMAC key are
// not the same bytes used for two different purposes.
const (
	encryptInfo = "distr/db/encrypt"
	macInfo     = "distr/db/hmac"
)

// MinKeyLen is the shortest accepted key. The underlying package derives its AES key with a plain
// SHA-256 hash rather than a password based KDF, so a key has to carry full entropy by itself.
const MinKeyLen = 32

// compressThreshold is the plaintext size above which compression is attempted. Ciphertext is
// incompressible, so Postgres can no longer TOAST-compress a column once it holds one, and
// compressing here is what keeps large values such as support bundle resources from growing.
const compressThreshold = 512

var (
	ErrNoKeys        = errors.New("no encryption key configured")
	ErrDuplicateKey  = errors.New("duplicate encryption key id")
	ErrKeyTooShort   = fmt.Errorf("encryption key must be at least %d bytes", MinKeyLen)
	ErrUnknownKey    = errors.New("value was encrypted with an unconfigured key")
	ErrUnknownFormat = errors.New("unrecognized encrypted value format")
	ErrUnknownFlag   = errors.New("unrecognized encrypted value flag")
	ErrTooShort      = errors.New("encrypted value too short")
)

type key struct {
	id  byte
	enc *crypto.Encryptor
	mac []byte
}

// Keyring holds every key an instance is configured with. The active key encrypts all writes; the
// others exist only to decrypt values written before a key rotation.
// A Keyring is safe for concurrent use by multiple goroutines.
type Keyring struct {
	active *key
	byID   map[byte]*key
	ids    []byte
}

// ParseKeyring reads the DATABASE_ENCRYPTION_KEY value, a comma separated list of `<id>:<base64 key>`
// entries whose first entry is the active key. A single entry may omit the ID, which then means 0.
func ParseKeyring(spec string) (*Keyring, error) {
	entries := strings.Split(spec, ",")
	keyring := Keyring{byID: make(map[byte]*key, len(entries))}
	for _, entry := range entries {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		k, err := parseKey(entry)
		if err != nil {
			return nil, err
		}
		if _, ok := keyring.byID[k.id]; ok {
			return nil, fmt.Errorf("%w: %d", ErrDuplicateKey, k.id)
		}
		keyring.byID[k.id] = k
		keyring.ids = append(keyring.ids, k.id)
		if keyring.active == nil {
			keyring.active = k
		}
	}
	if keyring.active == nil {
		return nil, ErrNoKeys
	}
	return &keyring, nil
}

func parseKey(entry string) (*key, error) {
	id := 0
	encoded := strings.TrimSpace(entry)
	if idPart, keyPart, ok := strings.Cut(encoded, ":"); ok {
		parsed, err := strconv.Atoi(strings.TrimSpace(idPart))
		if err != nil {
			return nil, fmt.Errorf("invalid encryption key id %q: %w", idPart, err)
		}
		if parsed < 0 || parsed > 255 {
			return nil, fmt.Errorf("encryption key id %d out of range 0-255", parsed)
		}
		id = parsed
		encoded = strings.TrimSpace(keyPart)
	}
	secret, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key with id %d: %w", id, err)
	}
	if len(secret) < MinKeyLen {
		return nil, fmt.Errorf("%w (id %d has %d)", ErrKeyTooShort, id, len(secret))
	}
	return newKey(byte(id), secret)
}

func newKey(id byte, secret []byte) (*key, error) {
	encKey, err := hkdf.Key(sha256.New, secret, nil, encryptInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("derive encryption subkey: %w", err)
	}
	macKey, err := hkdf.Key(sha256.New, secret, nil, macInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("derive mac subkey: %w", err)
	}
	encryptor, err := crypto.New(encKey)
	if err != nil {
		return nil, err
	}
	return &key{id: id, enc: encryptor, mac: macKey}, nil
}

// ActiveKeyID is the ID of the key that [Keyring.Encrypt] and [Keyring.HMAC] use.
func (k *Keyring) ActiveKeyID() byte { return k.active.id }

// KeyIDs are the IDs of every configured key, the active one first.
func (k *Keyring) KeyIDs() []byte { return slices.Clone(k.ids) }

// Encrypt seals plaintext with the active key.
func (k *Keyring) Encrypt(plaintext []byte) ([]byte, error) {
	flags := byte(0)
	payload := plaintext
	if compressed, ok := deflate(plaintext); ok {
		flags |= flagCompressed
		payload = compressed
	}
	sealed, err := k.active.enc.Encrypt(payload)
	if err != nil {
		return nil, err
	}
	return append([]byte{FormatVersion, k.active.id, flags}, sealed...), nil
}

// Decrypt opens a value produced by [Keyring.Encrypt], and returns a value that is still stored in a
// plaintext column unchanged.
func (k *Keyring) Decrypt(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return nil, ErrTooShort
	}
	switch value[0] {
	case plaintextMarker:
		// pgx may reuse the buffer a row was scanned from, so the result must not alias it.
		return bytes.Clone(value[1:]), nil
	case FormatVersion:
		break
	default:
		return nil, fmt.Errorf("%w: %d", ErrUnknownFormat, value[0])
	}
	if len(value) < headerLen {
		return nil, ErrTooShort
	}
	id, flags := value[1], value[2]
	key, ok := k.byID[id]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUnknownKey, id)
	}
	if flags & ^flagCompressed != 0 {
		return nil, fmt.Errorf("%w: %08b", ErrUnknownFlag, flags)
	}
	plaintext, err := key.enc.Decrypt(value[headerLen:])
	if err != nil {
		return nil, err
	}
	if flags&flagCompressed != 0 {
		return inflate(plaintext)
	}
	return plaintext, nil
}

// HMAC derives the lookup value of a credential that has to stay searchable, such as an access token,
// with the active key. Unlike [Keyring.Encrypt] it is deterministic and cannot be reversed, so a
// credential stored this way is neither guessable from a database dump nor recoverable by this
// instance.
func (k *Keyring) HMAC(value []byte) []byte {
	mac := hmac.New(sha256.New, k.active.mac)
	mac.Write(value)
	return mac.Sum(nil)
}

// HMACAll derives the lookup value under every configured key. A lookup has to accept any of them so
// that a credential stored before a key rotation is still found while the old key is configured.
func (k *Keyring) HMACAll(value []byte) [][]byte {
	all := make([][]byte, 0, len(k.ids))
	for _, id := range k.ids {
		mac := hmac.New(sha256.New, k.byID[id].mac)
		mac.Write(value)
		all = append(all, mac.Sum(nil))
	}
	return all
}

func deflate(plaintext []byte) ([]byte, bool) {
	if len(plaintext) < compressThreshold {
		return nil, false
	}
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, false
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, false
	}
	if err := w.Close(); err != nil {
		return nil, false
	}
	if buf.Len() >= len(plaintext) {
		return nil, false
	}
	return buf.Bytes(), true
}

func inflate(compressed []byte) ([]byte, error) {
	plaintext, err := io.ReadAll(flate.NewReader(bytes.NewReader(compressed)))
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}
	return plaintext, nil
}
