package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func mfaSetupHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	var buf bytes.Buffer
	if user, err := db.GetUserAccountByID(ctx, userID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return userAccountNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	} else if user.MFAEnabled {
		return respond.BadRequest("MFA is already enabled"), nil
	} else if key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Distr",
		AccountName: user.Email,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	}); err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	} else if err := db.UpdateUserAccountMFASecret(ctx, userID, dbcrypto.String(key.Secret())); err != nil {
		return nil, fmt.Errorf("failed to save MFA secret: %w", err)
	} else if img, err := key.Image(200, 200); err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	} else if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode QR code: %w", err)
	} else {
		qrCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

		return respond.JSON(api.SetupMFAResponse{
			Secret:    key.Secret(),
			QRCodeUrl: qrCode,
		}), nil
	}
}

func mfaEnableHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	if request, resp := decodeJSON[api.EnableMFARequest](r); resp != nil {
		return resp, nil
	} else if user, err := db.GetUserAccountByID(ctx, userID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return userAccountNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	} else if user.MFAEnabled {
		return respond.BadRequest("MFA is already enabled"), nil
	} else if user.MFASecret == nil {
		return respond.BadRequest("MFA not set up"), nil
	} else if valid := totp.Validate(request.Code, string(*user.MFASecret)); !valid {
		return respond.BadRequest("invalid code"), nil
	}

	codes, err := security.GenerateRecoveryCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	recoveryCodeRecords := make([]types.MFARecoveryCode, len(codes))
	for i, code := range codes {
		salt, hash, err := security.HashRecoveryCode(code)
		if err != nil {
			return nil, fmt.Errorf("failed to hash recovery code: %w", err)
		}
		recoveryCodeRecords[i] = types.MFARecoveryCode{
			CodeHash: hash,
			CodeSalt: salt,
		}
	}

	err = db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.EnableUserAccountMFA(ctx, userID); err != nil {
			return err
		}
		return db.CreateMFARecoveryCodes(ctx, userID, recoveryCodeRecords)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enable MFA: %w", err)
	}

	formattedCodes := make([]string, len(codes))
	for i, code := range codes {
		formattedCodes[i] = security.FormatRecoveryCode(code)
	}

	return respond.JSON(api.EnableMFAResponse{
		RecoveryCodes: formattedCodes,
	}), nil
}

func mfaDisableHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	if request, resp := decodeJSON[api.DisableMFARequest](r); resp != nil {
		return resp, nil
	} else if user, err := db.GetUserAccountByID(ctx, userID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return userAccountNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	} else if !user.MFAEnabled {
		return respond.BadRequest("MFA is not enabled"), nil
	} else if err := security.VerifyPassword(*user, request.Password); err != nil {
		return respond.Unauthorized("invalid password"), nil
	} else if err := db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.DisableUserAccountMFA(ctx, userID); err != nil {
			return err
		}
		return db.DeleteAllMFARecoveryCodes(ctx, userID)
	}); err != nil {
		return nil, fmt.Errorf("failed to disable MFA: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func mfaRegenerateRecoveryCodesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	if request, resp := decodeJSON[api.RegenerateMFARecoveryCodesRequest](r); resp != nil {
		return resp, nil
	} else if user, err := db.GetUserAccountByID(ctx, userID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return userAccountNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	} else if !user.MFAEnabled {
		return respond.BadRequest("MFA is not enabled"), nil
	} else if err := security.VerifyPassword(*user, request.Password); err != nil {
		return respond.Unauthorized("invalid password"), nil
	}

	codes, err := security.GenerateRecoveryCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	recoveryCodeRecords := make([]types.MFARecoveryCode, len(codes))
	for i, code := range codes {
		salt, hash, err := security.HashRecoveryCode(code)
		if err != nil {
			return nil, fmt.Errorf("failed to hash recovery code: %w", err)
		}
		recoveryCodeRecords[i] = types.MFARecoveryCode{
			CodeHash: hash,
			CodeSalt: salt,
		}
	}

	err = db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.DeleteAllMFARecoveryCodes(ctx, userID); err != nil {
			return err
		}
		return db.CreateMFARecoveryCodes(ctx, userID, recoveryCodeRecords)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to regenerate recovery codes: %w", err)
	}

	formattedCodes := make([]string, len(codes))
	for i, code := range codes {
		formattedCodes[i] = security.FormatRecoveryCode(code)
	}

	return respond.JSON(api.RegenerateMFARecoveryCodesResponse{
		RecoveryCodes: formattedCodes,
	}), nil
}

func mfaRecoveryCodesStatusHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	if user, err := db.GetUserAccountByID(ctx, userID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return userAccountNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
	} else if !user.MFAEnabled {
		return respond.BadRequest("MFA is not enabled"), nil
	} else if count, err := db.CountUnusedMFARecoveryCodes(ctx, userID); err != nil {
		return nil, fmt.Errorf("failed to count recovery codes: %w", err)
	} else {
		return respond.JSON(api.MFARecoveryCodesStatusResponse{
			RemainingCodes: count,
		}), nil
	}
}
