package types

import (
	"time"

	"github.com/google/uuid"
)

type MFARecoveryCode struct {
	ID            uuid.UUID  `db:"id"`
	CreatedAt     time.Time  `db:"created_at"`
	UserAccountID uuid.UUID  `db:"user_account_id"`
	CodeHash      []byte     `db:"code_hash"`
	CodeSalt      []byte     `db:"code_salt"`
	UsedAt        *time.Time `db:"used_at"`
}
