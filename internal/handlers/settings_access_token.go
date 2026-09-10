package handlers

import (
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
	accessTokenSecretSlotsExhaustedMessage = "This token already has two secrets. " +
		"Delete the one you want to replace before creating another."
	accessTokenLastSecretMessage = "This token must keep at least one secret. " +
		"Create the replacement first, or delete the token itself."
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

		if request.UserRole != nil {
			callerRole := auth.CurrentUserRole()
			if callerRole == nil || request.UserRole.GreaterThan(*callerRole) {
				http.Error(w, "token role cannot exceed your own role", http.StatusBadRequest)
				return
			}
		}

		newToken, err := authkey.NewToken()
		if err != nil {
			log.Warn("error creating token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		secret, err := newAccessTokenSecret(*newToken.Secret)
		if err != nil {
			log.Warn("error creating token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		token := types.AccessToken{
			ExpiresAt:      request.ExpiresAt,
			Label:          request.Label,
			UserAccountID:  auth.CurrentUserID(),
			Key:            newToken.Key,
			Secret1:        &secret,
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

		params := db.UpdateAccessTokenParams{
			UpdateLabel:     patch.Label.Present,
			Label:           patch.Label.Value,
			UpdateExpiresAt: patch.ExpiresAt.Present,
			ExpiresAt:       patch.ExpiresAt.Value,
			UpdateUserRole:  patch.UserRole.Present,
			UserRole:        patch.UserRole.Value,
		}
		if params.UserRole != nil {
			callerRole := auth.CurrentUserRole()
			if callerRole == nil || params.UserRole.GreaterThan(*callerRole) {
				http.Error(w, "token role cannot exceed your own role", http.StatusBadRequest)
				return
			}
		}

		updated, err := db.UpdateAccessToken(ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(), params)
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
		if slot == nil {
			http.Error(w, accessTokenSecretSlotsExhaustedMessage, http.StatusBadRequest)
			return
		}

		newSecret, err := authkey.NewSecret()
		if err != nil {
			log.Warn("error creating token secret", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		secret, err := newAccessTokenSecret(newSecret)
		if err != nil {
			log.Warn("error creating token secret", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		updated, err := db.CreateAccessTokenSecret(ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(), *slot, secret)
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, accessTokenSecretSlotsExhaustedMessage, http.StatusBadRequest)
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
		} else if token.Secret(slot.Other()) == nil {
			http.Error(w, accessTokenLastSecretMessage, http.StatusBadRequest)
			return
		}

		if err := db.DeleteAccessTokenSecret(
			ctx, tokenID, auth.CurrentUserID(), *auth.CurrentOrgID(), slot,
		); errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, accessTokenLastSecretMessage, http.StatusBadRequest)
		} else if err != nil {
			log.Warn("error deleting token secret", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func newAccessTokenSecret(key authkey.Secret) (types.AccessTokenSecret, error) {
	salt, err := authkey.NewSalt()
	if err != nil {
		return types.AccessTokenSecret{}, err
	}
	return types.AccessTokenSecret{Salt: salt, Hash: key.Hash(salt)}, nil
}
