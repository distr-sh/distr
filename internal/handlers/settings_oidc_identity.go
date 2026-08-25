package handlers

import (
	"errors"
	"net/http"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func getOIDCIdentitiesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	identities, err := db.GetUserAccountOIDCIdentities(ctx, auth.CurrentUserID())
	if err != nil {
		log.Warn("error getting OIDC identities", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.List(identities, mapping.UserAccountOIDCIdentityToAPI))
	}
}

func deleteOIDCIdentityHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	identityID, err := uuid.Parse(r.PathValue("oidcIdentityId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	auth := auth.Authentication.Require(ctx)
	user := auth.CurrentUser()

	// Accounts created via OIDC have no password, so disconnecting the only identity
	// would leave the user without any way to sign in.
	if len(user.PasswordHash) == 0 {
		if count, err := db.CountUserAccountOIDCIdentities(ctx, user.ID); err != nil {
			log.Warn("error counting OIDC identities", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if count <= 1 {
			http.Error(w,
				"this is the only way to sign in to your account. Set a password before disconnecting it",
				http.StatusConflict)
			return
		}
	}

	if err := db.DeleteUserAccountOIDCIdentity(ctx, user.ID, identityID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Warn("error deleting OIDC identity", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
