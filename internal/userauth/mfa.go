package userauth

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	"github.com/pquerna/otp/totp"
)

var (
	ErrMFARequired    = errors.New("mfa code required")
	ErrMFACodeInvalid = errors.New("invalid mfa code")
)

// VerifyMFA checks the given code against the user's TOTP secret and, if that does not match, against their
// unused recovery codes, marking a matching recovery code as used. It returns nil for a user without MFA, so
// every flow that hands out a session can call it unconditionally, ErrMFARequired when no code was given and
// ErrMFACodeInvalid when neither the TOTP code nor a recovery code matches.
func VerifyMFA(ctx context.Context, user types.UserAccount, code *string) error {
	if !user.MFAEnabled {
		return nil
	}
	if code == nil || *code == "" {
		return ErrMFARequired
	}
	if user.MFASecret == nil {
		// A database constraint guards against this, so it is a broken row rather than a failed verification.
		return errors.New("user has mfa enabled but no secret")
	}
	if totp.Validate(*code, string(*user.MFASecret)) {
		return nil
	}

	normalized := security.NormalizeRecoveryCode(*code)
	recoveryCodes, err := db.GetUnusedMFARecoveryCodes(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get recovery codes: %w", err)
	}
	for _, recoveryCode := range recoveryCodes {
		if security.VerifyRecoveryCode(normalized, recoveryCode.CodeSalt, recoveryCode.CodeHash) {
			return db.MarkMFARecoveryCodeAsUsed(ctx, recoveryCode.ID)
		}
	}
	return ErrMFACodeInvalid
}
