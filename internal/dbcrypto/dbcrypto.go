package dbcrypto

import (
	"fmt"
	"sync"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/util"
)

var keys = sync.OnceValues(func() (*Keyring, error) {
	return ParseKeyring(env.DatabaseEncryptionKey())
})

// Keys is the keyring of this instance.
func Keys() *Keyring { return util.Require(keys()) }

// Validate builds the keyring, so that a malformed DATABASE_ENCRYPTION_KEY aborts startup instead of
// failing the first query that touches an encrypted column.
func Validate() error {
	_, err := keys()
	return err
}

// TextValue renders the expression that reads an encrypted TEXT column. Until the encryption
// migration has run, a row may still hold its value in the plaintext column, so both are merged into
// one value that a [String] can scan. It carries no column alias, because an output expression is
// also used inside a row constructor, where an alias is a syntax error.
func TextValue(alias, column string) string {
	return fmt.Sprintf(
		`coalesce(%[1]s.%[2]s_enc, '\x00'::bytea || convert_to(%[1]s.%[2]s, 'UTF8'))`,
		alias, column,
	)
}

// BytesValue is [TextValue] for a BYTEA column, scannable into a [Bytes].
func BytesValue(alias, column string) string {
	return fmt.Sprintf(`coalesce(%[1]s.%[2]s_enc, '\x00'::bytea || %[1]s.%[2]s)`, alias, column)
}

// IsSetValue reports whether either representation of an encrypted column holds a value, for the
// booleans an API exposes in place of a secret it must not return.
func IsSetValue(alias, column string) string {
	return fmt.Sprintf(`num_nonnulls(%[1]s.%[2]s, %[1]s.%[2]s_enc) > 0`, alias, column)
}

// TextColumn is [TextValue] named after the column, for a result that is scanned by name.
func TextColumn(alias, column string) string {
	return TextValue(alias, column) + " AS " + column
}

// BytesColumn is [BytesValue] named after the column, for a result that is scanned by name.
func BytesColumn(alias, column string) string {
	return BytesValue(alias, column) + " AS " + column
}
