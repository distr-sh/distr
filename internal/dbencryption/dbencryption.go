// Package dbencryption moves values between the plaintext columns and the encrypted columns
// introduced by migration 130. It is the one-off counterpart to internal/dbcrypto, which encrypts
// and decrypts the values an already migrated instance reads and writes.
package dbencryption

import (
	"context"
	"errors"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"go.uber.org/zap"
)

// RunEncrypt encrypts every value that is still stored in a plaintext column and re-encrypts every
// value that still uses a key that is no longer active, which is what lets a retired key be removed
// from the keyring. It is safe to run repeatedly and while the server is serving traffic, and it can
// be interrupted and resumed.
func RunEncrypt(ctx context.Context) error {
	log := internalctx.GetLogger(ctx)
	var errs []error
	var encrypted, reencrypted int64
	for _, column := range db.EncryptedColumns {
		rows, err := db.EncryptPlaintextRows(ctx, column)
		encrypted += rows
		if err != nil {
			log.Error("could not encrypt column", zap.Stringer("column", column), zap.Error(err))
			errs = append(errs, err)
			continue
		}
		if rows > 0 {
			log.Info("encrypted column", zap.Stringer("column", column), zap.Int64("rows", rows))
		}

		// Selecting the rows of a retired key cannot use an index, so the indexed check comes
		// first: with no rotation to finish, it saves a scan that reads every encrypted value.
		stale, err := db.HasStaleKeyRows(ctx, column)
		if err != nil {
			log.Error("could not check for values of a retired key",
				zap.Stringer("column", column), zap.Error(err))
			errs = append(errs, err)
			continue
		} else if !stale {
			continue
		}

		rows, err = db.ReencryptStaleKeyRows(ctx, column)
		reencrypted += rows
		if err != nil {
			log.Error("could not re-encrypt column", zap.Stringer("column", column), zap.Error(err))
			errs = append(errs, err)
			continue
		}
		if rows > 0 {
			log.Info("re-encrypted column under the active key",
				zap.Stringer("column", column), zap.Int64("rows", rows))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	log.Info("database encryption complete",
		zap.Int64("encrypted", encrypted), zap.Int64("reencrypted", reencrypted))
	return nil
}

// RunDecrypt writes every encrypted value back into its plaintext column, which is what makes the
// rollback of migration 130 keep the data instead of discarding it. It is the last thing a
// downgrade to a version that does not encrypt runs while the new schema is still in place, and the
// server has to be stopped for it: the server writes only encrypted columns, so it would both undo
// this and keep it from ever finishing.
func RunDecrypt(ctx context.Context) error {
	log := internalctx.GetLogger(ctx)
	log.Warn("moving every encrypted value back into its plaintext column. " +
		"Stop Distr before this runs and do not start it again until the database is migrated down")
	var errs []error
	var decrypted int64
	for _, column := range db.EncryptedColumns {
		rows, err := db.DecryptEncryptedRows(ctx, column)
		decrypted += rows
		if err != nil {
			log.Error("could not decrypt column", zap.Stringer("column", column), zap.Error(err))
			errs = append(errs, err)
			continue
		}
		if rows > 0 {
			log.Info("decrypted column", zap.Stringer("column", column), zap.Int64("rows", rows))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	log.Info("database decryption complete", zap.Int64("decrypted", decrypted))
	return nil
}

// WarnAboutUnencrypted logs which columns still hold values in plaintext or under a retired key, so
// that an instance that has never run the encryption migration, or has rotated its key without
// finishing the rotation, says so on every start instead of silently carrying them.
func WarnAboutUnencrypted(ctx context.Context) {
	log := internalctx.GetLogger(ctx)
	var plaintext, staleKey []string
	for _, column := range db.EncryptedColumns {
		if found, err := db.HasPlaintextRows(ctx, column); err != nil {
			log.Warn("could not check for unencrypted values", zap.Stringer("column", column), zap.Error(err))
		} else if found {
			plaintext = append(plaintext, column.String())
		}
		if found, err := db.HasStaleKeyRows(ctx, column); err != nil {
			log.Warn("could not check for values of a retired key", zap.Stringer("column", column), zap.Error(err))
		} else if found {
			staleKey = append(staleKey, column.String())
		}
	}
	if len(plaintext) > 0 {
		log.Warn(
			"the database still holds unencrypted values. Run `distr maintenance encrypt-database`, "+
				"or set DATABASE_ENCRYPTION_MIGRATE_ON_BOOT=true to encrypt them during startup",
			zap.Strings("columns", plaintext),
		)
	}
	if len(staleKey) > 0 {
		log.Warn(
			"the database still holds values encrypted with a key that is no longer active. Keep that key "+
				"in DATABASE_ENCRYPTION_KEY and run `distr maintenance encrypt-database` to finish the rotation",
			zap.Strings("columns", staleKey),
		)
	}
}
