package db

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// SealKind is how the value of an EncryptedColumn is protected.
type SealKind int

const (
	// SealEncrypt is reversible, so a value stored this way can be read back and re-sealed under a
	// new key.
	SealEncrypt SealKind = iota
	// SealHMAC is a keyed hash. It keeps a credential searchable, and is one-way: a value stored this
	// way survives a key rotation only while the key it was hashed with is still configured, and the
	// credential has to be reissued to move it to a new key.
	SealHMAC
)

// EncryptedColumn is one column whose value moved from a plaintext column into an encrypted or keyed
// hash column in migration 129. Rows written before that migration still hold their value in the
// plaintext column until EncryptPlaintextRows has moved it over.
type EncryptedColumn struct {
	Table  string
	Column string
	// Target is the column the sealed value is written to.
	Target string
	Kind   SealKind
	// Binary is true when the plaintext column is a BYTEA rather than a TEXT.
	Binary bool
	// BatchSize is how many rows are read into memory at once. It is small for the tables whose rows
	// can be large.
	BatchSize int
}

// EncryptedColumns is every column the encryption migration covers, in the order it processes them.
var EncryptedColumns = []EncryptedColumn{
	{Table: "Secret", Column: "value", Target: "value_enc"},
	{Table: "CustomOIDCConfiguration", Column: "client_secret", Target: "client_secret_enc"},
	{Table: "CustomEmailConfiguration", Column: "smtp_password", Target: "smtp_password_enc"},
	{Table: "Artifact", Column: "upstream_username", Target: "upstream_username_enc"},
	{Table: "Artifact", Column: "upstream_password", Target: "upstream_password_enc"},
	{Table: "UserAccount", Column: "mfa_secret", Target: "mfa_secret_enc"},
	{Table: "Organization", Column: "stripe_webhook_secret", Target: "stripe_webhook_secret_enc"},
	{Table: "ApplicationEntitlement", Column: "registry_password", Target: "registry_password_enc"},
	{Table: "SupportBundle", Column: "bundle_secret", Target: "bundle_secret_enc"},
	{Table: "AccessToken", Column: "key", Target: "key_hmac", Kind: SealHMAC, Binary: true},
	{
		Table: "DeploymentRevision", Column: "values_yaml", Target: "values_yaml_enc",
		Binary: true, BatchSize: 200,
	},
	{
		Table: "DeploymentRevision", Column: "env_file_data", Target: "env_file_data_enc",
		Binary: true, BatchSize: 200,
	},
	{Table: "SupportBundleResource", Column: "content", Target: "content_enc", BatchSize: 50},
}

func (c EncryptedColumn) String() string { return c.Table + "." + c.Column }

const defaultEncryptionBatchSize = 500

func (c EncryptedColumn) batchSize() int {
	if c.BatchSize > 0 {
		return c.BatchSize
	}
	return defaultEncryptionBatchSize
}

// plaintextExpr reads the plaintext column as bytes, so that a TEXT and a BYTEA column can be sealed
// by the same code and produce exactly what the application writes.
func (c EncryptedColumn) plaintextExpr() string {
	if c.Binary {
		return c.Column
	}
	return fmt.Sprintf("convert_to(%s, 'UTF8')", c.Column)
}

func (c EncryptedColumn) seal(keyring *dbcrypto.Keyring, plaintext []byte) ([]byte, error) {
	if c.Kind == SealHMAC {
		return keyring.HMAC(plaintext), nil
	}
	return keyring.Encrypt(plaintext)
}

// staleKeyCond matches rows whose ciphertext was sealed with a key that is no longer the active one.
// The key id is the second byte of the stored value, which is what makes this answerable in SQL
// without decrypting anything.
const staleKeyCond = "get_byte(%[1]s, 0) = %[2]d AND get_byte(%[1]s, 1) <> %[3]d"

func (c EncryptedColumn) staleKeyExpr() string {
	return fmt.Sprintf(staleKeyCond, c.Target, dbcrypto.FormatVersion, dbcrypto.Keys().ActiveKeyID())
}

func exists(ctx context.Context, query string) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, "SELECT EXISTS("+query+")")
	if err != nil {
		return false, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
}

// HasPlaintextRows reports whether any row still holds its value in the plaintext column.
func HasPlaintextRows(ctx context.Context, c EncryptedColumn) (bool, error) {
	found, err := exists(ctx, fmt.Sprintf("SELECT 1 FROM %s WHERE %s IS NOT NULL", c.Table, c.Column))
	if err != nil {
		return false, fmt.Errorf("could not check %v for plaintext rows: %w", c, err)
	}
	return found, nil
}

// HasStaleKeyRows reports whether any row is still sealed with a key that is no longer active. It is
// always false for a keyed hash, which cannot be moved to another key.
func HasStaleKeyRows(ctx context.Context, c EncryptedColumn) (bool, error) {
	if c.Kind == SealHMAC {
		return false, nil
	}
	found, err := exists(ctx,
		fmt.Sprintf("SELECT 1 FROM %s WHERE %s IS NOT NULL AND %s", c.Table, c.Target, c.staleKeyExpr()))
	if err != nil {
		return false, fmt.Errorf("could not check %v for rows of a retired key: %w", c, err)
	}
	return found, nil
}

type sealedRow struct {
	ID    uuid.UUID `db:"id"`
	Value []byte    `db:"value"`
}

// EncryptPlaintextRows seals every row of one column that is still stored in plaintext and returns
// how many rows it rewrote.
func EncryptPlaintextRows(ctx context.Context, c EncryptedColumn) (int64, error) {
	return rewrite(ctx, c,
		fmt.Sprintf("%s AS value FROM %s WHERE %s IS NOT NULL", c.plaintextExpr(), c.Table, c.Column),
		fmt.Sprintf("%s IS NOT NULL", c.Column),
		c.seal,
	)
}

// ReencryptStaleKeyRows re-seals every row of one column that was encrypted with a key that is no
// longer active, which is what lets a retired key be removed from the keyring. It does nothing for a
// keyed hash: the value behind it cannot be recovered, so an access token created under a retired
// key has to be reissued instead.
func ReencryptStaleKeyRows(ctx context.Context, c EncryptedColumn) (int64, error) {
	if c.Kind == SealHMAC {
		return 0, nil
	}
	return rewrite(ctx, c,
		fmt.Sprintf("%s AS value FROM %s WHERE %s IS NOT NULL AND %s",
			c.Target, c.Table, c.Target, c.staleKeyExpr()),
		c.staleKeyExpr(),
		func(keyring *dbcrypto.Keyring, value []byte) ([]byte, error) {
			plaintext, err := keyring.Decrypt(value)
			if err != nil {
				return nil, err
			}
			return keyring.Encrypt(plaintext)
		},
	)
}

// rewrite reads the rows selected by from in batches, re-seals each one, and writes the result to
// the target column. Each batch is a statement of its own, so the work can be interrupted and
// resumed, and it can run while the server is serving traffic: guard repeats the selection criteria
// in the update, which makes it a no-op for a row someone else has rewritten since it was read.
func rewrite(
	ctx context.Context,
	c EncryptedColumn,
	from, guard string,
	seal func(*dbcrypto.Keyring, []byte) ([]byte, error),
) (int64, error) {
	db := internalctx.GetDb(ctx)
	keyring := dbcrypto.Keys()
	var total int64
	for {
		rows, err := db.Query(ctx, fmt.Sprintf("SELECT id, %s LIMIT %d", from, c.batchSize()))
		if err != nil {
			return total, fmt.Errorf("could not query rows of %v: %w", c, err)
		}
		batch, err := pgx.CollectRows(rows, pgx.RowToStructByName[sealedRow])
		if err != nil {
			return total, fmt.Errorf("could not collect rows of %v: %w", c, err)
		}
		if len(batch) == 0 {
			return total, nil
		}

		ids := make([]uuid.UUID, len(batch))
		sealed := make([][]byte, len(batch))
		for i, row := range batch {
			ids[i] = row.ID
			if sealed[i], err = seal(keyring, row.Value); err != nil {
				return total, fmt.Errorf("could not seal %v of row %v: %w", c, row.ID, err)
			}
		}

		tag, err := db.Exec(ctx,
			fmt.Sprintf(
				`UPDATE %[1]s AS t SET %[2]s = NULL, %[3]s = v.sealed
				FROM (SELECT unnest(@ids::UUID[]) AS id, unnest(@sealed::BYTEA[]) AS sealed) v
				WHERE t.id = v.id AND %[4]s`,
				c.Table, c.Column, c.Target, guard),
			pgx.NamedArgs{"ids": ids, "sealed": sealed},
		)
		if err != nil {
			return total, fmt.Errorf("could not seal rows of %v: %w", c, err)
		}
		total += tag.RowsAffected()

		if len(batch) < c.batchSize() {
			return total, nil
		}
	}
}
