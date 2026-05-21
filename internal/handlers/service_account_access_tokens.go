package handlers

import (
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/authkey"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ServiceAccountAccessTokensRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Service Account Tokens"))

	r.Get("/", listServiceAccountAccessTokensHandler).
		With(option.Description("List access tokens for a service account")).
		With(option.Request(api.ServiceAccountIDRequest{})).
		With(option.Response(http.StatusOK, []api.AccessToken{}))

	r.Post("/", createServiceAccountAccessTokenHandler).
		With(option.Description("Create a new access token for a service account")).
		With(option.Request(struct {
			api.ServiceAccountIDRequest
			api.CreateServiceAccountAccessTokenRequest
		}{})).
		With(option.Response(http.StatusCreated, api.AccessTokenWithKey{}))

	r.Delete("/{tokenId}", deleteServiceAccountAccessTokenHandler).
		With(option.Description("Delete an access token of a service account")).
		With(option.Request(api.ServiceAccountAccessTokenIDRequest{}))
}

func listServiceAccountAccessTokensHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	sa := internalctx.GetServiceAccount(ctx)

	tokens, err := db.GetServiceAccountAccessTokens(ctx, sa.ID)
	if err != nil {
		log.Error("failed to list service account tokens", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, mapping.List(tokens, mapping.ServiceAccountAccessTokenToDTO))
}

func createServiceAccountAccessTokenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	sa := internalctx.GetServiceAccount(ctx)

	body, err := JsonBody[api.CreateServiceAccountAccessTokenRequest](w, r)
	if err != nil {
		return
	}

	key, err := authkey.NewKey()
	if err != nil {
		log.Warn("error creating token key", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token := types.ServiceAccountAccessToken{
		ExpiresAt:        body.ExpiresAt,
		Label:            body.Label,
		ServiceAccountID: sa.ID,
		Key:              key,
	}
	if err := db.CreateServiceAccountAccessToken(ctx, &token); err != nil {
		log.Warn("error creating service account token", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	RespondJSON(w, mapping.ServiceAccountAccessTokenToDTO(token).WithKey(token.Key))
}

func deleteServiceAccountAccessTokenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	sa := internalctx.GetServiceAccount(ctx)

	tokenID, err := uuid.Parse(r.PathValue("tokenId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := db.DeleteServiceAccountAccessToken(ctx, tokenID, sa.ID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Warn("error deleting service account token", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
