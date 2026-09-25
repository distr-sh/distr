package db

import (
	"context"
	"fmt"
	"strings"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EncryptedColumn is one column whose value moved from a plaintext column into an encrypted column
// in migration 130. Rows written before that migration still hold their value in the plaintext
// column until EncryptPlaintextRows has moved it over.
type EncryptedColumn struct {
	dbcrypto.Column
	// Binary is true when the plaintext column is a BYTEA rather than a TEXT.
	Binary bool
	// BatchSize is how many rows are read into memory at once. It is small for the tables whose rows
	// can be large.
	BatchSize int
}

func encrypted(table, name string, scope ...string) EncryptedColumn {
	return EncryptedColumn{Column: dbcrypto.NewColumn(table, name, scope...)}
}

func (c EncryptedColumn) binary() EncryptedColumn {
	c.Binary = true
	return c
}

func (c EncryptedColumn) batched(size int) EncryptedColumn {
	c.BatchSize = size
	return c
}

// Every encrypted column of the schema, with the scope its values are bound to. A column that is
// missing here is skipped by the encryption migration, by the rollback and by the startup warning,
// so it is declared once and referenced from the queries that read and write it, rather than named
// again at every call site. Organization and UserAccount are scoped to their own id because that id
// is the owner an authorization check reads.
var (
	secretValue = encrypted("Secret", "value",
		"customer_organization_id", "organization_id")
	oidcClientSecret = encrypted("CustomOIDCConfiguration", "client_secret",
		"custom_domain_id", "organization_id")
	emailSMTPUsername           = encrypted("CustomEmailConfiguration", "smtp_username", "organization_id")
	emailSMTPPassword           = encrypted("CustomEmailConfiguration", "smtp_password", "organization_id")
	artifactUpstreamUsername    = encrypted("Artifact", "upstream_username", "organization_id")
	artifactUpstreamPassword    = encrypted("Artifact", "upstream_password", "organization_id")
	userAccountMFASecret        = encrypted("UserAccount", "mfa_secret", "id")
	organizationStripeSecret    = encrypted("Organization", "stripe_webhook_secret", "id")
	entitlementRegistryUsername = encrypted("ApplicationEntitlement", "registry_username",
		"customer_organization_id", "organization_id")
	entitlementRegistryPassword = encrypted("ApplicationEntitlement", "registry_password",
		"customer_organization_id", "organization_id")
	supportBundleSecret = encrypted("SupportBundle", "bundle_secret",
		"customer_organization_id", "organization_id")
	deploymentValuesYaml = encrypted("DeploymentRevision", "values_yaml", "deployment_id").
				binary().batched(200)
	deploymentEnvFileData = encrypted("DeploymentRevision", "env_file_data", "deployment_id").
				binary().batched(200)
	supportBundleResourceContent = encrypted("SupportBundleResource", "content", "support_bundle_id").
					batched(50)
)

// EncryptedColumns is every column the encryption migration covers, in the order it processes them.
var EncryptedColumns = []EncryptedColumn{
	secretValue,
	oidcClientSecret,
	emailSMTPUsername,
	emailSMTPPassword,
	artifactUpstreamUsername,
	artifactUpstreamPassword,
	userAccountMFASecret,
	organizationStripeSecret,
	entitlementRegistryUsername,
	entitlementRegistryPassword,
	supportBundleSecret,
	deploymentValuesYaml,
	deploymentEnvFileData,
	supportBundleResourceContent,
}

// Output renders the expression that reads this column, named after it for a scan by name.
func (c EncryptedColumn) Output(alias string) string {
	if c.Binary {
		return c.BytesColumn(alias)
	}
	return c.TextColumn(alias)
}

// Value is [EncryptedColumn.Output] without the column alias, for a row constructor where an alias
// is a syntax error.
func (c EncryptedColumn) Value(alias string) string {
	if c.Binary {
		return c.BytesValue(alias)
	}
	return c.TextValue(alias)
}

const defaultEncryptionBatchSize = 500

func (c EncryptedColumn) batchSize() int {
	if c.BatchSize > 0 {
		return c.BatchSize
	}
	return defaultEncryptionBatchSize
}

// plaintextExpr reads the plaintext column as bytes, so that a TEXT and a BYTEA column can be
// encrypted by the same code and produce exactly what the application writes.
func (c EncryptedColumn) plaintextExpr() string {
	if c.Binary {
		return c.Name
	}
	return fmt.Sprintf("convert_to(%s, 'UTF8')", c.Name)
}

// decryptedExpr is the inverse of plaintextExpr: it writes decrypted bytes back into the plaintext
// column, in whichever type that column has.
func (c EncryptedColumn) decryptedExpr() string {
	if c.Binary {
		return "v.rewritten"
	}
	return "convert_from(v.rewritten, 'UTF8')"
}

// keyPrefixExpr reads the format version and key id a value was sealed with. It has to stay
// identical to the expression migration 130 indexes, and uses substring rather than get_byte
// because that reads a TOAST slice instead of the whole value. The data a value is bound to is
// authenticated rather than stored, so it does not appear here.
func (c EncryptedColumn) keyPrefixExpr() string {
	return fmt.Sprintf("substring(%s FROM 1 FOR 2)", c.Enc())
}

// activeKeyPrefix is the literal that keyPrefixExpr yields for a value sealed with the active key.
func activeKeyPrefix() string {
	return fmt.Sprintf(`'\x%02x%02x'::BYTEA`, dbcrypto.FormatVersion, dbcrypto.Keys().ActiveKeyID())
}

func (c EncryptedColumn) staleKeyExpr() string {
	return fmt.Sprintf("%s <> %s", c.keyPrefixExpr(), activeKeyPrefix())
}

func queryBool(ctx context.Context, query string) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, query)
	if err != nil {
		return false, err
	}
	return pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
}

// HasPlaintextRows reports whether any row still holds its value in the plaintext column.
func HasPlaintextRows(ctx context.Context, c EncryptedColumn) (bool, error) {
	found, err := queryBool(ctx,
		fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s IS NOT NULL)", c.Table, c.Name))
	if err != nil {
		return false, fmt.Errorf("could not check %v for plaintext rows: %w", c, err)
	}
	return found, nil
}

// HasStaleKeyRows reports whether any row is still encrypted with a key that is no longer active.
// min and max are answered from the index of migration 130, while a search for a mismatch cannot
// be: no index serves <>, so every plan for it reads the value of every row.
func HasStaleKeyRows(ctx context.Context, c EncryptedColumn) (bool, error) {
	found, err := queryBool(ctx, fmt.Sprintf(
		`SELECT coalesce(min(%[1]s) <> %[2]s OR max(%[1]s) <> %[2]s, false)
		FROM %[3]s WHERE %[4]s IS NOT NULL`,
		c.keyPrefixExpr(), activeKeyPrefix(), c.Table, c.Enc()))
	if err != nil {
		return false, fmt.Errorf("could not check %v for rows of a retired key: %w", c, err)
	}
	return found, nil
}

type encryptedRow struct {
	ID    uuid.UUID   `db:"id"`
	Scope []uuid.UUID `db:"scope"`
	Value []byte      `db:"value"`
}

// scopeExpr reads the scope of a row in the order the column is bound to it.
func (c EncryptedColumn) scopeExpr(alias string) string {
	values := make([]string, len(c.Scope))
	for i, scope := range c.Scope {
		values[i] = c.ScopeValue(alias, scope)
	}
	return "ARRAY[" + strings.Join(values, ", ") + "]::UUID[]"
}

// EncryptPlaintextRows encrypts every row of one column that is still stored in plaintext and
// returns how many rows it rewrote.
func EncryptPlaintextRows(ctx context.Context, c EncryptedColumn) (int64, error) {
	return rewrite(ctx, c,
		c.plaintextExpr(),
		fmt.Sprintf("%s IS NOT NULL", c.Name),
		fmt.Sprintf("%s = NULL, %s = v.rewritten", c.Name, c.Enc()),
		func(value []byte, scope []uuid.UUID) ([]byte, error) { return c.EncryptRaw(value, scope...) },
	)
}

// DecryptEncryptedRows moves every encrypted row of one column back into its plaintext column. It is
// the inverse of EncryptPlaintextRows and, unlike it, must not run while the server is serving:
// every write of an encrypted column produces a new ciphertext, so a row rewritten between the read
// and the update here would lose that write, and a column the server keeps writing to never empties.
func DecryptEncryptedRows(ctx context.Context, c EncryptedColumn) (int64, error) {
	return rewrite(ctx, c,
		c.Enc(),
		fmt.Sprintf("%s IS NOT NULL", c.Enc()),
		fmt.Sprintf("%s = NULL, %s = %s", c.Enc(), c.Name, c.decryptedExpr()),
		func(value []byte, scope []uuid.UUID) ([]byte, error) { return c.Decrypt(value, scope...) },
	)
}

// ReencryptStaleKeyRows re-encrypts every row of one column that was encrypted with a key that is no
// longer active, which is what lets a retired key be removed from the keyring.
func ReencryptStaleKeyRows(ctx context.Context, c EncryptedColumn) (int64, error) {
	return rewrite(ctx, c,
		c.Enc(),
		fmt.Sprintf("%s IS NOT NULL AND %s", c.Enc(), c.staleKeyExpr()),
		fmt.Sprintf("%s = v.rewritten", c.Enc()),
		func(value []byte, scope []uuid.UUID) ([]byte, error) {
			plaintext, err := c.Decrypt(value, scope...)
			if err != nil {
				return nil, err
			}
			return c.EncryptRaw(plaintext, scope...)
		},
	)
}

// rewrite reads the rows selected by where in batches, transforms each value and applies set to the
// row it came from. Each batch is a statement of its own, so the work can be interrupted and
// resumed, and where is repeated in the update, which makes it a no-op for a row someone else has
// rewritten since it was read.
//
// Batches resume at the id of the last one, because no index answers a search for a key that is not
// the active one, so a rotation would otherwise rescan everything it has already rewritten. Nothing
// falls behind the cursor and back into the selection: every write seals with the active key.
//
// The value is named rewritten because Secret has a column called value, which would make every
// unqualified reference to it in set ambiguous.
func rewrite(
	ctx context.Context,
	c EncryptedColumn,
	valueExpr, where, set string,
	transform func([]byte, []uuid.UUID) ([]byte, error),
) (int64, error) {
	db := internalctx.GetDb(ctx)
	var total int64
	var cursor uuid.UUID
	for {
		rows, err := db.Query(ctx,
			fmt.Sprintf(
				`SELECT id, %s AS scope, %s AS value FROM %s AS t
				WHERE %s AND id > @cursor ORDER BY id LIMIT %d`,
				c.scopeExpr("t"), valueExpr, c.Table, where, c.batchSize()),
			pgx.NamedArgs{"cursor": cursor})
		if err != nil {
			return total, fmt.Errorf("could not query rows of %v: %w", c, err)
		}
		batch, err := pgx.CollectRows(rows, pgx.RowToStructByName[encryptedRow])
		if err != nil {
			return total, fmt.Errorf("could not collect rows of %v: %w", c, err)
		}
		if len(batch) == 0 {
			return total, nil
		}
		cursor = batch[len(batch)-1].ID

		ids := make([]uuid.UUID, len(batch))
		rewritten := make([][]byte, len(batch))
		for i, row := range batch {
			ids[i] = row.ID
			if rewritten[i], err = transform(row.Value, row.Scope); err != nil {
				return total, fmt.Errorf("could not rewrite %v of row %v: %w", c, row.ID, err)
			}
		}

		tag, err := db.Exec(ctx,
			fmt.Sprintf(
				`UPDATE %[1]s AS t SET %[2]s
				FROM (SELECT unnest(@ids::UUID[]) AS id, unnest(@rewritten::BYTEA[]) AS rewritten) v
				WHERE t.id = v.id AND %[3]s`,
				c.Table, set, where),
			pgx.NamedArgs{"ids": ids, "rewritten": rewritten},
		)
		if err != nil {
			return total, fmt.Errorf("could not rewrite rows of %v: %w", c, err)
		}
		total += tag.RowsAffected()

		if len(batch) < c.batchSize() {
			return total, nil
		}
	}
}
