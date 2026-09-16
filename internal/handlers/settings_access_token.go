package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authkey"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	accessTokenCredentialsExhaustedMessage = "This access token already has two tokens. " +
		"Delete the one you want to replace before adding another."
	accessTokenLastCredentialMessage = "This access token must keep at least one token. " +
		"Add the replacement first, or delete the access token itself."
	accessTokenRoleExceedsCallerMessage = "token role cannot exceed your own role"
)

func getAccessTokensHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		tokens, err := db.GetAccessTokens(ctx, auth.CurrentUserID(), *auth.CurrentOrgID())
		if err != nil {
			log.Warn("error getting tokens", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.List(tokens, mapping.AccessTokenToAPI))
		}
	}
}

func createAccessTokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		request, err := JsonBody[api.CreateAccessTokenRequest](w, r)
		if err != nil {
			return
		}

		if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if !checkAccessTokenRole(ctx, w, request.UserRole) {
			return
		}

		newToken, err := authkey.NewToken()
		if err != nil {
			log.Warn("error creating token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		token := types.AccessToken{
			Label:         request.Label,
			UserAccountID: auth.CurrentUserID(),
			Key:           newToken.Key,
			Secret1: &types.AccessTokenSecret{
				Hash:      newToken.Secret.Hash(),
				ExpiresAt: request.ExpiresAt,
			},
			OrganizationID: *auth.CurrentOrgID(),
			UserRole:       request.UserRole,
		}
		if err := db.CreateAccessToken(ctx, &token); err != nil {
			log.Warn("error creating token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSONWithStatus(w, http.StatusCreated, mapping.AccessTokenToAPI(token).WithKey(newToken.Serialize()))
		}
	}
}

func patchAccessTokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		tokenID, err := uuid.Parse(r.PathValue("accessTokenId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		patch, err := JsonBody[api.PatchAccessTokenRequest](w, r)
		if err != nil {
			return
		}

		updated, err := db.UpdateAccessTokenLabel(
			ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(), patch.Label,
		)
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if err != nil {
			log.Warn("error updating token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.AccessTokenToAPI(*updated))
		}
	}
}

func deleteAccessTokenHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		tokenID, err := uuid.Parse(r.PathValue("accessTokenId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		auth := auth.Authentication.Require(ctx)
		if err := db.DeleteAccessToken(ctx, tokenID, auth.CurrentUserID()); err != nil {
			log.Warn("error deleting token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func createAccessTokenSecretHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		tokenID, err := uuid.Parse(r.PathValue("accessTokenId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		request, err := JsonBody[api.CreateAccessTokenSecretRequest](w, r)
		if err != nil {
			return
		}

		if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		token, err := db.GetAccessToken(ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Warn("error getting token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slot := token.FreeSecretSlot()
		if slot == nil || token.CredentialCount() >= 2 {
			http.Error(w, accessTokenCredentialsExhaustedMessage, http.StatusBadRequest)
			return
		}

		newSecret, err := authkey.NewSecret()
		if err != nil {
			log.Warn("error creating token secret", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		secret := types.AccessTokenSecret{Hash: newSecret.Hash(), ExpiresAt: request.ExpiresAt}
		updated, err := db.CreateAccessTokenSecret(ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(), *slot, secret)
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, accessTokenCredentialsExhaustedMessage, http.StatusBadRequest)
		} else if err != nil {
			log.Warn("error creating token secret", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			rotated := authkey.Token{Key: updated.Key, Secret: &newSecret}
			RespondJSONWithStatus(
				w,
				http.StatusCreated,
				mapping.AccessTokenToAPI(*updated).WithKey(rotated.Serialize()),
			)
		}
	}
}

func deleteAccessTokenSecretHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		tokenID, err := uuid.Parse(r.PathValue("accessTokenId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		slotValue, err := strconv.Atoi(r.PathValue("slot"))
		slot := types.AccessTokenSecretSlot(slotValue)
		if err != nil || !slot.Valid() {
			http.NotFound(w, r)
			return
		}

		token, err := db.GetAccessToken(ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Warn("error getting token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if token.Secret(slot) == nil {
			http.NotFound(w, r)
			return
		} else if token.CredentialCount() < 2 {
			http.Error(w, accessTokenLastCredentialMessage, http.StatusBadRequest)
			return
		}

		if err := db.DeleteAccessTokenSecret(
			ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(), slot,
		); errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, accessTokenLastCredentialMessage, http.StatusBadRequest)
		} else if err != nil {
			log.Warn("error deleting token secret", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// deleteAccessTokenKeyHandler retires the key as a credential of its own, which is the last step of
// migrating a token issued before secrets existed.
func deleteAccessTokenKeyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		tokenID, err := uuid.Parse(r.PathValue("accessTokenId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		token, err := db.GetAccessToken(ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) || (err == nil && !token.KeyIsCredential) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Warn("error getting token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if !token.HasSecrets() {
			http.Error(w, accessTokenLastCredentialMessage, http.StatusBadRequest)
			return
		}

		if err := db.RetireAccessTokenKeyCredential(
			ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(),
		); errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, accessTokenLastCredentialMessage, http.StatusBadRequest)
		} else if err != nil {
			log.Warn("error retiring token key", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func checkAccessTokenRole(ctx context.Context, w http.ResponseWriter, requested *types.UserRole) bool {
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	membershipRole, err := db.GetUserRoleInOrganization(ctx, auth.CurrentUserID(), *auth.CurrentOrgID())
	if err != nil {
		log.Warn("error getting user role", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if !types.AccessTokenRoleAllowed(requested, auth.CurrentUserRole(), membershipRole) {
		http.Error(w, accessTokenRoleExceedsCallerMessage, http.StatusBadRequest)
		return false
	}
	return true
}
