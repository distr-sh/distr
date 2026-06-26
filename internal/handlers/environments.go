package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/featureflags"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func EnvironmentsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Environments"))
	r.With(
		middleware.RequireVendor,
		middleware.RequireOrgAndRole,
		middleware.ExperimentalFeatureFlagMiddleware(featureflags.KeyEnvironments),
	).Group(func(r chiopenapi.Router) {
		r.Get("/", getEnvironmentsHandler()).
			With(option.Description("List environments")).
			With(option.Response(http.StatusOK, []api.Environment{}))

		r.Route("/{environmentId}", func(r chiopenapi.Router) {
			type EnvironmentIDRequest struct {
				EnvironmentID uuid.UUID `path:"environmentId"`
			}

			r.Get("/", getEnvironmentHandler()).
				With(option.Description("Get an environment")).
				With(option.Request(EnvironmentIDRequest{})).
				With(option.Response(http.StatusOK, api.Environment{}))

			r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
				r.Put("/", updateEnvironmentHandler()).
					With(option.Description("Update an environment")).
					With(option.Request(struct {
						EnvironmentIDRequest
						api.CreateUpdateEnvironmentRequest
					}{})).
					With(option.Response(http.StatusOK, api.Environment{}))

				r.Delete("/", deleteEnvironmentHandler()).
					With(option.Description("Delete an environment")).
					With(option.Request(EnvironmentIDRequest{}))
			})
		})

		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", createEnvironmentHandler()).
			With(option.Description("Create an environment")).
			With(option.Request(api.CreateUpdateEnvironmentRequest{})).
			With(option.Response(http.StatusOK, api.Environment{}))
	})
}

func getEnvironmentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		environments, err := db.GetEnvironmentsByOrganizationID(ctx, *auth.CurrentOrgID())
		if err != nil {
			log.Error("failed to get environments", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		RespondJSON(w, environmentResponses(environments))
	}
}

func getEnvironmentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("environmentId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		environment, err := db.GetEnvironment(ctx, id, *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if err != nil {
			log.Error("failed to get environment", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSON(w, mapping.EnvironmentToAPI(*environment))
		}
	}
}

func createEnvironmentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.CreateUpdateEnvironmentRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		environment := environmentFromCreateUpdateRequest(*auth.CurrentOrgID(), request)
		if err := db.CreateEnvironment(ctx, &environment); err != nil {
			handleEnvironmentWriteError(w, r, log, "create", err)
			return
		}
		RespondJSON(w, mapping.EnvironmentToAPI(environment))
	}
}

func updateEnvironmentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("environmentId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.CreateUpdateEnvironmentRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		environment := environmentFromCreateUpdateRequest(*auth.CurrentOrgID(), request)
		environment.ID = id
		if err := db.UpdateEnvironment(ctx, &environment); err != nil {
			handleEnvironmentWriteError(w, r, log, "update", err)
			return
		}
		RespondJSON(w, mapping.EnvironmentToAPI(environment))
	}
}

//nolint:dupl
func deleteEnvironmentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("environmentId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		if err := db.DeleteEnvironmentWithID(ctx, id, *auth.CurrentOrgID()); errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "environment is in use", http.StatusConflict)
		} else if err != nil {
			log.Error("failed to delete environment", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

func environmentFromCreateUpdateRequest(orgID uuid.UUID, request api.CreateUpdateEnvironmentRequest) types.Environment {
	return types.Environment{
		OrganizationID:      orgID,
		Name:                strings.TrimSpace(request.Name),
		Description:         request.Description,
		SortOrder:           request.SortOrder,
		IsProduction:        request.IsProduction,
		AllowDynamicTargets: request.AllowDynamicTargets,
		RetentionPolicyID:   request.RetentionPolicyID,
	}
}

func environmentResponses(environments []types.Environment) []api.Environment {
	return mapping.List(environments, mapping.EnvironmentToAPI)
}

func handleEnvironmentWriteError(w http.ResponseWriter, r *http.Request, log *zap.Logger, action string, err error) {
	if errors.Is(err, apierrors.ErrAlreadyExists) {
		http.Error(w, "an environment with this name already exists", http.StatusBadRequest)
	} else if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else {
		log.Error("failed to "+action+" environment", zap.Error(err))
		sentry.GetHubFromContext(r.Context()).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
