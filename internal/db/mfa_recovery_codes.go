package db

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func CreateMFARecoveryCodes(ctx context.Context, userID uuid.UUID, codes []types.MFARecoveryCode) error {
	db := internalctx.GetDb(ctx)

	for _, code := range codes {
		_, err := db.Exec(ctx,
			`INSERT INTO UserAccount_MFARecoveryCode
             (user_account_id, code_hash, code_salt)
             VALUES (@user_account_id, @code_hash, @code_salt)`,
			pgx.NamedArgs{
				"user_account_id": userID,
				"code_hash":       code.CodeHash,
				"code_salt":       code.CodeSalt,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to insert recovery code: %w", err)
		}
	}

	return nil
}

func GetUnusedMFARecoveryCodes(ctx context.Context, userID uuid.UUID) ([]types.MFARecoveryCode, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT id, created_at, user_account_id, code_hash, code_salt, used_at
         FROM UserAccount_MFARecoveryCode
         WHERE user_account_id = @user_account_id AND used_at IS NULL
         ORDER BY created_at`,
		pgx.NamedArgs{
			"user_account_id": userID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query recovery codes: %w", err)
	}

	codes, err := pgx.CollectRows(rows, pgx.RowToStructByPos[types.MFARecoveryCode])
	if err != nil {
		return nil, fmt.Errorf("failed to collect recovery codes: %w", err)
	}

	return codes, nil
}

func CountUnusedMFARecoveryCodes(ctx context.Context, userID uuid.UUID) (int, error) {
	db := internalctx.GetDb(ctx)
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM UserAccount_MFARecoveryCode
         WHERE user_account_id = @user_account_id AND used_at IS NULL`,
		pgx.NamedArgs{
			"user_account_id": userID,
		},
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count recovery codes: %w", err)
	}
	return count, nil
}

func MarkMFARecoveryCodeAsUsed(ctx context.Context, codeID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx,
		`UPDATE UserAccount_MFARecoveryCode
         SET used_at = now()
         WHERE id = @id AND used_at IS NULL`,
		pgx.NamedArgs{
			"id": codeID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to mark recovery code as used: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("recovery code not found or already used")
	}
	return nil
}

func DeleteAllMFARecoveryCodes(ctx context.Context, userID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	_, err := db.Exec(ctx,
		`DELETE FROM UserAccount_MFARecoveryCode WHERE user_account_id = @user_account_id`,
		pgx.NamedArgs{
			"user_account_id": userID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to delete recovery codes: %w", err)
	}
	return nil
}
