package handlers

import (
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func UpdateNotificationConfigurationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Notifications"))

	r.Use(middleware.ProFeature)

	r.Get("/", getUpdateNotificationConfigurationsHandler()).
		With(option.Description("list all update notification configurations")).
		With(option.Request(customerScopeQuery{})).
		With(option.Response(http.StatusOK, []api.UpdateNotificationConfiguration{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Post("/", createUpdateNotificationConfigurationHandler()).
		With(option.Description("create a new update notification configuration")).
		With(option.Request(api.CreateUpdateNotificationConfigurationRequest{})).
		With(option.Response(http.StatusOK, api.UpdateNotificationConfiguration{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Route("/{id}", func(r chiopenapi.Router) {
			r.Put("/", updateUpdateNotificationConfigurationHandler()).
				With(option.Description("update an existing update notification configuration")).
				With(option.Request(struct {
					notificationConfigurationIDRequest
					api.CreateUpdateNotificationConfigurationRequest
				}{})).
				With(option.Response(http.StatusOK, api.UpdateNotificationConfiguration{}))

			r.Delete("/", deleteUpdateNotificationConfigurationHandler()).
				With(option.Description("delete an existing update notification configuration")).
				With(option.Request(struct {
					notificationConfigurationIDRequest
					customerScopeQuery
				}{}))
		})
}

type notificationConfigurationIDRequest struct {
	ID string `path:"id"`
}

func getUpdateNotificationConfigurationsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		customerOrgID, ok := resolveCustomerScopeFromQuery(w, r)
		if !ok {
			return
		}

		configs, err := db.GetUpdateNotificationConfigurations(ctx, *auth.CurrentOrgID(), customerOrgID)
		if err != nil {
			respondInternalError(w, r, err, "failed to get update notification configurations")
			return
		}

		RespondJSON(w, mapping.List(configs, mapping.UpdateNotificationConfigurationToAPI))
	}
}

func createUpdateNotificationConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		request, ok := updateNotificationConfigurationBody(w, r)
		if !ok {
			return
		}

		customerOrgID, ok := resolveCustomerScope(w, r, request.CustomerOrganizationID)
		if !ok {
			return
		}

		config := mapping.UpdateNotificationConfigurationToInternal(request, *auth.CurrentOrgID(), customerOrgID)
		if err := db.CreateUpdateNotificationConfiguration(ctx, &config); err != nil {
			respondInternalError(w, r, err, "failed to create update notification configuration")
			return
		}

		RespondJSON(w, mapping.UpdateNotificationConfigurationToAPI(config))
	}
}

func updateUpdateNotificationConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		request, ok := updateNotificationConfigurationBody(w, r)
		if !ok {
			return
		}

		customerOrgID, ok := resolveCustomerScope(w, r, request.CustomerOrganizationID)
		if !ok {
			return
		}

		config := mapping.UpdateNotificationConfigurationToInternal(request, *auth.CurrentOrgID(), customerOrgID)
		config.ID = id
		if err := db.UpdateUpdateNotificationConfiguration(ctx, &config); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				respondInternalError(w, r, err, "failed to save update notification configuration")
			}
			return
		}

		RespondJSON(w, mapping.UpdateNotificationConfigurationToAPI(config))
	}
}

func deleteUpdateNotificationConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)

		id, err := uuid.Parse(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		customerOrgID, ok := resolveCustomerScopeFromQuery(w, r)
		if !ok {
			return
		}

		if err := db.DeleteUpdateNotificationConfiguration(ctx, id, *auth.CurrentOrgID(), customerOrgID); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				respondInternalError(w, r, err, "failed to delete update notification configuration")
			}
			return
		}
	}
}

func updateNotificationConfigurationBody(
	w http.ResponseWriter,
	r *http.Request,
) (api.CreateUpdateNotificationConfigurationRequest, bool) {
	request, err := JsonBody[api.CreateUpdateNotificationConfigurationRequest](w, r)
	if err != nil {
		return request, false
	} else if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return request, false
	}
	return request, true
}
