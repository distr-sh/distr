package handlers

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func SecretsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Secrets"))

	r.Use(middleware.RequireOrgAndRole)

	r.Get("/", getSecretsHandler()).
		With(option.Description("List all secrets")).
		With(option.Response(http.StatusOK, []api.SecretWithoutValue{}))

	r.Group(func(r chiopenapi.Router) {
		r.Use(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin)

		r.Post("/", createSecretHandler()).
			With(option.Description("Create a secret")).
			With(option.Request(api.CreateSecretRequest{})).
			With(option.Response(http.StatusCreated, api.SecretWithoutValue{}))

		r.Route("/{secretId}", func(r chiopenapi.Router) {
			r.Put("/", updateSecretHandler()).
				With(option.Description("Update a secret")).
				With(option.Request(api.UpdateSecretRequest{})).
				With(option.Response(http.StatusOK, api.SecretWithoutValue{}))

			r.Delete("/", deleteSecretHandler()).
				With(option.Description("Delete a secret")).
				With(option.Request(api.DeleteSecretRequest{}))
		})
	})
}

func getSecretsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		secrets, err := db.GetSecrets(ctx, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID())

		if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get secrets", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.List(secrets, mapping.SecretToAPI))
		}
	}
}

func createSecretHandler() http.HandlerFunc {
	secretKeyPattern := regexp.MustCompile(`^[a-zA-Z][\w_]*$`)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)
		body, err := JsonBody[api.CreateSecretRequest](w, r)
		if err != nil {
			return
		}

		if body.Key == "" {
			http.Error(w, "key is required", http.StatusBadRequest)
			return
		}

		if !secretKeyPattern.MatchString(body.Key) {
			http.Error(w, "invalid key format", http.StatusBadRequest)
			return
		}

		if body.Value == "" {
			http.Error(w, "value is required", http.StatusBadRequest)
			return
		}
		if customerOrganizationID := auth.CurrentCustomerOrgID(); customerOrganizationID != nil {
			if body.CustomerOrganizationID != nil && *body.CustomerOrganizationID != *customerOrganizationID {
				http.Error(w, "customer organization ID mismatch", http.StatusBadRequest)
				return
			}
			body.CustomerOrganizationID = customerOrganizationID
		}

		secret, err := db.CreateSecret(
			ctx,
			*auth.CurrentOrgID(),
			body.CustomerOrganizationID,
			auth.CurrentUserID(),
			body.Key,
			body.Value,
		)

		if err != nil {
			if errors.Is(err, apierrors.ErrConflict) {
				http.Error(w, "secret with this key already exists", http.StatusConflict)
			} else {
				internalctx.GetLogger(ctx).Error("failed to create secret", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusCreated)
			RespondJSON(w, mapping.SecretToAPI(*secret))
		}
	}
}

func updateSecretHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)
		id, err := uuid.Parse(r.PathValue("secretId"))
		if err != nil {
			http.Error(w, "invalid secret ID", http.StatusBadRequest)
			return
		}

		body, err := JsonBody[api.UpdateSecretRequest](w, r)
		if err != nil {
			return
		}

		// check if this user is authorized to update the secret
		_, err = db.GetSecretByID(ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID())
		if err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.Error(w, "secret not found", http.StatusNotFound)
				return
			}
			internalctx.GetLogger(ctx).Error("failed to get secret", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		secret, err := db.UpdateSecret(
			ctx,
			id,
			auth.CurrentCustomerOrgID(),
			auth.CurrentUserID(),
			body.Value,
		)

		if err != nil {
			internalctx.GetLogger(ctx).Error("failed to update secret", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.SecretToAPI(*secret))
		}
	}
}

func deleteSecretHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)
		id, err := uuid.Parse(r.PathValue("secretId"))
		if err != nil {
			http.Error(w, "invalid secret ID", http.StatusBadRequest)
			return
		}

		err = db.DeleteSecret(ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID())
		if err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.Error(w, "secret not found", http.StatusNotFound)
			} else {
				internalctx.GetLogger(ctx).Error("failed to delete secret", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}
