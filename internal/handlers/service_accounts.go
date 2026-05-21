package handlers

import (
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ServiceAccountsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Service Accounts"))
	r.Use(
		middleware.RequireOrgAndRole,
		middleware.RequireAdmin,
		middleware.BlockSuperAdmin,
		middleware.BlockServiceAccount,
	)

	r.Get("/", listServiceAccountsHandler).
		With(option.Description("List all service accounts")).
		With(option.Response(http.StatusOK, []api.ServiceAccountResponse{}))

	r.Post("/", createServiceAccountHandler).
		With(option.Description("Create a new service account")).
		With(option.Request(api.CreateServiceAccountRequest{})).
		With(option.Response(http.StatusCreated, api.ServiceAccountResponse{}))

	r.Route("/{serviceAccountId}", func(r chiopenapi.Router) {
		r.Use(serviceAccountMiddleware)

		r.Get("/", getServiceAccountHandler).
			With(option.Description("Get a service account")).
			With(option.Request(api.ServiceAccountIDRequest{})).
			With(option.Response(http.StatusOK, api.ServiceAccountResponse{}))

		r.Patch("/", patchServiceAccountHandler).
			With(option.Description("Update a service account")).
			With(option.Request(struct {
				api.ServiceAccountIDRequest
				api.PatchServiceAccountRequest
			}{})).
			With(option.Response(http.StatusOK, api.ServiceAccountResponse{}))

		r.Delete("/", deleteServiceAccountHandler).
			With(option.Description("Delete a service account")).
			With(option.Request(api.ServiceAccountIDRequest{}))

		r.Route("/tokens", ServiceAccountAccessTokensRouter)
	})
}

func listServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	var sas []types.ServiceAccount
	var err error
	if customerOrgID := auth.CurrentCustomerOrgID(); customerOrgID != nil {
		sas, err = db.GetServiceAccountsByCustomerOrgID(ctx, *customerOrgID)
	} else {
		sas, err = db.GetServiceAccountsByOrgID(ctx, *auth.CurrentOrgID())
	}
	if err != nil {
		log.Error("failed to list service accounts", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, mapping.List(sas, mapping.ServiceAccountToAPI))
}

func createServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	body, err := JsonBody[api.CreateServiceAccountRequest](w, r)
	if err != nil {
		return
	}

	if _, err := types.ParseAccountRole(string(body.AccountRole)); err != nil {
		http.Error(w, "invalid accountRole", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	customerOrgID := body.CustomerOrganizationID
	if callerCustomerOrgID := auth.CurrentCustomerOrgID(); callerCustomerOrgID != nil {
		if customerOrgID != nil && *customerOrgID != *callerCustomerOrgID {
			http.Error(w, "cannot create service account outside your customer organization", http.StatusForbidden)
			return
		}
		customerOrgID = callerCustomerOrgID
	} else if customerOrgID != nil {
		co, err := db.GetCustomerOrganizationByID(ctx, *customerOrgID)
		if errors.Is(err, apierrors.ErrNotFound) || (err == nil && co.OrganizationID != *auth.CurrentOrgID()) {
			http.Error(w, "customer organization does not exist", http.StatusBadRequest)
			return
		} else if err != nil {
			log.Error("failed to get customer organization", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	sa := types.ServiceAccount{
		OrganizationID:         *auth.CurrentOrgID(),
		CustomerOrganizationID: customerOrgID,
		Name:                   body.Name,
		AccountRole:            body.AccountRole,
	}
	if err := db.CreateServiceAccount(ctx, &sa); err != nil {
		if errors.Is(err, apierrors.ErrAlreadyExists) {
			http.Error(w, "a service account with this name already exists", http.StatusConflict)
			return
		}
		log.Error("failed to create service account", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	RespondJSON(w, mapping.ServiceAccountToAPI(sa))
}

func getServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	sa := internalctx.GetServiceAccount(r.Context())
	RespondJSON(w, mapping.ServiceAccountToAPI(*sa))
}

func patchServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	sa := internalctx.GetServiceAccount(ctx)

	body, err := JsonBody[api.PatchServiceAccountRequest](w, r)
	if err != nil {
		return
	}

	if body.Name != nil {
		if *body.Name == "" {
			http.Error(w, "name cannot be empty", http.StatusBadRequest)
			return
		}
		sa.Name = *body.Name
	}
	if body.AccountRole != nil {
		if _, err := types.ParseAccountRole(string(*body.AccountRole)); err != nil {
			http.Error(w, "invalid accountRole", http.StatusBadRequest)
			return
		}
		sa.AccountRole = *body.AccountRole
	}

	if err := db.UpdateServiceAccount(ctx, sa); err != nil {
		if errors.Is(err, apierrors.ErrAlreadyExists) {
			http.Error(w, "a service account with this name already exists", http.StatusConflict)
			return
		} else if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Error("failed to update service account", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, mapping.ServiceAccountToAPI(*sa))
}

func deleteServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	sa := internalctx.GetServiceAccount(ctx)

	if err := db.DeleteServiceAccount(ctx, sa.ID, *auth.CurrentOrgID()); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Error("failed to delete service account", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func serviceAccountMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)
		log := internalctx.GetLogger(ctx)

		saID, err := uuid.Parse(r.PathValue("serviceAccountId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		sa, err := db.GetServiceAccountByID(ctx, saID, *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Warn("error getting service account", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// A customer admin may only act on SAs scoped to their own customer org.
		if callerCustomerOrgID := auth.CurrentCustomerOrgID(); callerCustomerOrgID != nil {
			if sa.CustomerOrganizationID == nil || *sa.CustomerOrganizationID != *callerCustomerOrgID {
				http.NotFound(w, r)
				return
			}
		}
		h.ServeHTTP(w, r.WithContext(internalctx.WithServiceAccount(ctx, sa)))
	})
}
