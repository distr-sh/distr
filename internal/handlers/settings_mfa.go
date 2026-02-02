package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/png"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"go.uber.org/zap"
)

func mfaSetupHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	user, err := db.GetUserAccountByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
		} else {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to get user", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if user.MFAEnabled {
		http.Error(w, "MFA is already enabled", http.StatusBadRequest)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Distr",
		AccountName: user.Email,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to generate TOTP key", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := db.UpdateUserAccountMFASecret(ctx, userID, key.Secret()); err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to save MFA secret", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	img, err := key.Image(200, 200)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to generate QR code", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to encode QR code", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	qrCode := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	RespondJSON(w, api.SetupMFAResponse{
		Secret:    key.Secret(),
		QRCodeUrl: qrCode,
	})
}

func mfaEnableHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	request, err := JsonBody[api.EnableMFARequest](w, r)
	if err != nil {
		return
	}

	user, err := db.GetUserAccountByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
		} else {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to get user", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if user.MFAEnabled {
		http.Error(w, "MFA is already enabled", http.StatusBadRequest)
		return
	}

	if user.MFASecret == nil {
		http.Error(w, "MFA not set up", http.StatusBadRequest)
		return
	}

	valid := totp.Validate(request.Code, *user.MFASecret)
	if !valid {
		http.Error(w, "invalid code", http.StatusBadRequest)
		return
	}

	codes, err := security.GenerateRecoveryCodes()
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to generate recovery codes", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	recoveryCodeRecords := make([]types.MFARecoveryCode, len(codes))
	for i, code := range codes {
		salt, hash, err := security.HashRecoveryCode(code)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to hash recovery code", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
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
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to enable MFA", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	formattedCodes := make([]string, len(codes))
	for i, code := range codes {
		formattedCodes[i] = security.FormatRecoveryCode(code)
	}

	RespondJSON(w, api.EnableMFAResponse{
		RecoveryCodes: formattedCodes,
	})
}

func mfaDisableHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	request, err := JsonBody[api.DisableMFARequest](w, r)
	if err != nil {
		return
	}

	user, err := db.GetUserAccountByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
		} else {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to get user", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if !user.MFAEnabled {
		http.Error(w, "MFA is not enabled", http.StatusBadRequest)
		return
	}

	if err := security.VerifyPassword(*user, request.Password); err != nil {
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	err = db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.DisableUserAccountMFA(ctx, userID); err != nil {
			return err
		}
		return db.DeleteAllMFARecoveryCodes(ctx, userID)
	})
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to disable MFA", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func mfaRegenerateRecoveryCodesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	request, err := JsonBody[api.RegenerateMFARecoveryCodesRequest](w, r)
	if err != nil {
		return
	}

	user, err := db.GetUserAccountByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
		} else {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to get user", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if !user.MFAEnabled {
		http.Error(w, "MFA is not enabled", http.StatusBadRequest)
		return
	}

	if err := security.VerifyPassword(*user, request.Password); err != nil {
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	codes, err := security.GenerateRecoveryCodes()
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to generate recovery codes", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	recoveryCodeRecords := make([]types.MFARecoveryCode, len(codes))
	for i, code := range codes {
		salt, hash, err := security.HashRecoveryCode(code)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to hash recovery code", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
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
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to regenerate recovery codes", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	formattedCodes := make([]string, len(codes))
	for i, code := range codes {
		formattedCodes[i] = security.FormatRecoveryCode(code)
	}

	RespondJSON(w, api.RegenerateMFARecoveryCodesResponse{
		RecoveryCodes: formattedCodes,
	})
}

func mfaRecoveryCodesStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authInfo := auth.Authentication.Require(ctx)
	userID := authInfo.CurrentUserID()

	user, err := db.GetUserAccountByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
		} else {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to get user", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if !user.MFAEnabled {
		http.Error(w, "MFA is not enabled", http.StatusBadRequest)
		return
	}

	count, err := db.CountUnusedMFARecoveryCodes(ctx, userID)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to count recovery codes", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	RespondJSON(w, api.MFARecoveryCodesStatusResponse{
		RemainingCodes: count,
	})
}
