package dbcrypto

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/glasskube/pkg/crypto"
	"github.com/google/uuid"
)

// FormatVersion is the first byte of every encrypted value, and the key id is the second. A
// maintenance query relies on this to find the values of a retired key without decrypting them.
const FormatVersion = crypto.FormatVersion

// plaintextMarker is the additional authenticated data length of a value that a query read from the
// plaintext column of a row the encryption migration has not moved over yet, which is how a read
// tells the two apart without a second column in the result. Every encrypted value is bound to at
// least the name of its column, so a length of zero can only mean this.
const plaintextMarker byte = 0

// scopeLen is the length of the row scope an encrypted value is bound to, which is always a UUID.
// Because it is fixed, the column name and the scope can be concatenated without a separator.
const scopeLen = len(uuid.UUID{})

// ErrValueTooShort is returned for a value that is too short to hold the additional authenticated
// data its length prefix announces, which means the read expression and this code disagree.
var ErrValueTooShort = errors.New("encrypted value is shorter than its authenticated data")

var keys *crypto.Keyring

// Init parses the given DATABASE_ENCRYPTION_KEY value into the keyring of this instance. Every
// command that reads or writes an encrypted column has to call it before it does, so that a
// malformed key aborts startup instead of failing the first query that touches such a column. The
// key is passed in because internal/types imports this package, and through it every agent binary,
// which has no internal/env.
func Init(spec string) error {
	keyring, err := crypto.ParseKeyring(spec)
	if err != nil {
		return err
	}
	keys = keyring
	return nil
}

// Keys MUST be called after [Init], otherwise it WILL panic.
func Keys() *crypto.Keyring {
	if keys == nil {
		panic("detected call to dbcrypto.Keys before calling dbcrypto.Init")
	}
	return keys
}

// Column is one encrypted column. Every value it holds is authenticated along with the column it is
// stored in and the scope of its row, so a value copied elsewhere, or a row handed to another owner,
// no longer decrypts. Both are part of the stored format: renaming either, or changing
// [Column.Scope], invalidates every value in the column until it has been re-encrypted.
type Column struct {
	Table string
	Name  string
	// Scope names the columns of the same row that an authorization check reads to decide who the
	// value belongs to, most specific first. Binding the owner rather than the row id is what stops
	// someone who can write the database from claiming a row, since the owner is stored in the clear
	// next to the ciphertext. A statement that changes one of them has to rewrite the encrypted
	// column too, or the value it leaves behind cannot be opened.
	Scope []string
}

func NewColumn(table, name string, scope ...string) Column {
	return Column{Table: table, Name: name, Scope: scope}
}

func (c Column) String() string { return c.Table + "." + c.Name }

// Enc is the column the ciphertext is stored in, next to the plaintext column that rows written
// before the encryption migration still use.
func (c Column) Enc() string { return c.Name + "_enc" }

// ScopeOf maps a nullable scope column to the nil UUID the read expression substitutes for a NULL.
// No row carries that value, so the two cannot collide.
func ScopeOf(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}

// aad is never stored: a read reconstructs it from the column and the row it reads. Every scope
// value has the same length, so concatenating them is unambiguous.
func (c Column) aad(scope []uuid.UUID) ([]byte, error) {
	if len(scope) != len(c.Scope) {
		return nil, fmt.Errorf("%v is bound to %v, but %d scope values were given",
			c, c.Scope, len(scope))
	}
	aad := make([]byte, 0, len(c.String())+len(scope)*scopeLen)
	aad = append(aad, c.String()...)
	for _, value := range scope {
		aad = append(aad, value[:]...)
	}
	return aad, nil
}

// Encrypt seals a value of this column for a row of the given scope. Compression is on because
// Postgres can no longer TOAST-compress a column once it holds ciphertext, and the columns this
// package covers include support bundle resources and deployment revision values, which are large
// enough for that to matter.
func (c Column) Encrypt(value String, scope ...uuid.UUID) ([]byte, error) {
	return c.EncryptRaw([]byte(value), scope...)
}

// EncryptPtr seals a nullable [String], keeping nil as NULL.
func (c Column) EncryptPtr(value *String, scope ...uuid.UUID) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return c.Encrypt(*value, scope...)
}

// EncryptBytes seals a [Bytes], keeping nil as NULL, because a nullable column of this kind uses
// NULL to mean that there is no value rather than that there is an empty one.
func (c Column) EncryptBytes(value Bytes, scope ...uuid.UUID) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return c.EncryptRaw(value, scope...)
}

// EncryptRaw seals bytes that are neither a [String] nor a [Bytes], which only the encryption
// migration has, since it moves a TEXT and a BYTEA column with the same code.
func (c Column) EncryptRaw(value []byte, scope ...uuid.UUID) ([]byte, error) {
	aad, err := c.aad(scope)
	if err != nil {
		return nil, err
	}
	return Keys().Encrypt(value, crypto.WithCompression(), crypto.WithAAD(aad))
}

// Decrypt opens a value read straight from the encrypted column, which only the encryption migration
// does. Everything else reads through [Column.TextColumn] and its siblings and decrypts while
// scanning.
func (c Column) Decrypt(value []byte, scope ...uuid.UUID) ([]byte, error) {
	aad, err := c.aad(scope)
	if err != nil {
		return nil, err
	}
	return Keys().DecryptWithAAD(value, aad)
}

// TextValue renders the expression that reads this column as a TEXT column, scannable into a
// [String]. Until the encryption migration has run, a row may still hold its value in the plaintext
// column, so both are merged into one value that tells the two apart by its first byte. It carries
// no column alias, because an output expression is also used inside a row constructor, where an
// alias is a syntax error.
func (c Column) TextValue(alias string) string {
	return fmt.Sprintf(`coalesce(%s, '\x00'::BYTEA || convert_to(%s.%s, 'UTF8'))`,
		c.boundValue(alias), alias, c.Name)
}

// BytesValue is [Column.TextValue] for a BYTEA column, scannable into a [Bytes].
func (c Column) BytesValue(alias string) string {
	return fmt.Sprintf(`coalesce(%s, '\x00'::BYTEA || %s.%s)`, c.boundValue(alias), alias, c.Name)
}

// TextColumn is [Column.TextValue] named after the column, for a result that is scanned by name.
func (c Column) TextColumn(alias string) string {
	return c.TextValue(alias) + " AS " + c.Name
}

// BytesColumn is [Column.BytesValue] named after the column, for a result that is scanned by name.
func (c Column) BytesColumn(alias string) string {
	return c.BytesValue(alias) + " AS " + c.Name
}

// IsSetValue reports whether either representation of the column holds a value, for the booleans an
// API exposes in place of a secret it must not return.
func (c Column) IsSetValue(alias string) string {
	return fmt.Sprintf(`num_nonnulls(%[1]s.%[2]s, %[1]s.%[3]s) > 0`, alias, c.Name, c.Enc())
}

// ScopeValue reads one scope column, substituting the nil UUID for a NULL. See [ScopeOf].
func (c Column) ScopeValue(alias, scope string) string {
	return fmt.Sprintf(`coalesce(%s.%s, '%s'::UUID)`, alias, scope, uuid.Nil)
}

// boundValue is the stored value prefixed with the length and the bytes of its additional
// authenticated data, which is what lets a scan reconstruct it from bytes alone. The whole
// expression is NULL when the encrypted column is, so it can be the first argument of a coalesce
// that falls back to the plaintext column.
func (c Column) boundValue(alias string) string {
	name := []byte(c.String())
	aadLen := len(name) + len(c.Scope)*scopeLen
	if aadLen > 255 {
		panic("dbcrypto: additional authenticated data of " + c.String() + " does not fit its length prefix")
	}
	expr := fmt.Sprintf(`'\x%02x%x'::BYTEA`, aadLen, name)
	for _, scope := range c.Scope {
		expr += fmt.Sprintf(` || uuid_send(%s)`, c.ScopeValue(alias, scope))
	}
	return expr + fmt.Sprintf(` || %s.%s`, alias, c.Enc())
}

// decryptScanned opens what [Column.TextValue] and its siblings read: either a value that is still
// stored in a plaintext column, or a ciphertext preceded by the data it was bound to.
func decryptScanned(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return nil, ErrValueTooShort
	}
	if value[0] == plaintextMarker {
		// pgx may reuse the buffer a row was scanned from, so the result must not alias it.
		return bytes.Clone(value[1:]), nil
	}
	aadLen := int(value[0])
	if len(value) < 1+aadLen {
		return nil, ErrValueTooShort
	}
	return Keys().DecryptWithAAD(value[1+aadLen:], value[1:1+aadLen])
}
