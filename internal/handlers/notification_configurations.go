package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func UpdateNotificationConfigurationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Notifications"))

	r.Use(middleware.ProFeature, middleware.UpdateNotificationsCustomerFeatureMiddleware)

	r.Get("/", respond.Handler(getUpdateNotificationConfigurationsHandler)).
		With(option.Description("list all update notification configurations")).
		With(option.Request(customerScopeQuery{})).
		With(option.Response(http.StatusOK, []api.UpdateNotificationConfiguration{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Post("/", respond.Handler(createUpdateNotificationConfigurationHandler)).
		With(option.Description("create a new update notification configuration")).
		With(option.Request(api.CreateUpdateNotificationConfigurationRequest{})).
		With(option.Response(http.StatusOK, api.UpdateNotificationConfiguration{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Route("/{id}", func(r chiopenapi.Router) {
			r.Put("/", respond.Handler(updateUpdateNotificationConfigurationHandler)).
				With(option.Description("update an existing update notification configuration")).
				With(option.Request(struct {
					notificationConfigurationIDRequest
					api.CreateUpdateNotificationConfigurationRequest
				}{})).
				With(option.Response(http.StatusOK, api.UpdateNotificationConfiguration{}))

			r.Delete("/", respond.Handler(deleteUpdateNotificationConfigurationHandler)).
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

var updateNotificationConfigurationNotFound = respond.NotFound("update notification configuration not found")

func getUpdateNotificationConfigurationsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if customerOrgID, resp, err := customerScopeFromQuery(r); resp != nil || err != nil {
		return resp, err
	} else if configs, err := db.GetUpdateNotificationConfigurations(
		ctx, *auth.CurrentOrgID(), customerOrgID,
	); err != nil {
		return nil, fmt.Errorf("failed to get update notification configurations: %w", err)
	} else {
		return respond.JSON(mapping.List(configs, mapping.UpdateNotificationConfigurationToAPI)), nil
	}
}

func createUpdateNotificationConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if request, resp := updateNotificationConfigurationBody(r); resp != nil {
		return resp, nil
	} else if customerOrgID, resp, err := customerScope(ctx, request.CustomerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else {
		config := mapping.UpdateNotificationConfigurationToInternal(request, *auth.CurrentOrgID(), customerOrgID)
		if err := db.CreateUpdateNotificationConfiguration(ctx, &config); err != nil {
			return nil, fmt.Errorf("failed to create update notification configuration: %w", err)
		} else {
			return respond.JSON(mapping.UpdateNotificationConfigurationToAPI(config)), nil
		}
	}
}

func updateUpdateNotificationConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("id")); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if request, resp := updateNotificationConfigurationBody(r); resp != nil {
		return resp, nil
	} else if customerOrgID, resp, err := customerScope(ctx, request.CustomerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else {
		config := mapping.UpdateNotificationConfigurationToInternal(request, *auth.CurrentOrgID(), customerOrgID)
		config.ID = id
		if err := db.UpdateUpdateNotificationConfiguration(ctx, &config); err != nil {
			if errors.Is(err, distrerr.ErrNotFound) {
				return updateNotificationConfigurationNotFound, nil
			} else {
				return nil, fmt.Errorf("failed to save update notification configuration: %w", err)
			}
		} else {
			return respond.JSON(mapping.UpdateNotificationConfigurationToAPI(config)), nil
		}
	}
}

func deleteUpdateNotificationConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("id")); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if customerOrgID, resp, err := customerScopeFromQuery(r); resp != nil || err != nil {
		return resp, err
	} else if err := db.DeleteUpdateNotificationConfiguration(ctx, id, *auth.CurrentOrgID(), customerOrgID); err != nil {
		if errors.Is(err, distrerr.ErrNotFound) {
			return updateNotificationConfigurationNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to delete update notification configuration: %w", err)
		}
	} else {
		return respond.Status(http.StatusOK), nil
	}
}

func updateNotificationConfigurationBody(
	r *http.Request,
) (api.CreateUpdateNotificationConfigurationRequest, respond.Responder) {
	if request, resp := decodeJSON[api.CreateUpdateNotificationConfigurationRequest](r); resp != nil {
		return request, resp
	} else if err := request.Validate(); err != nil {
		return request, respond.BadRequest(err.Error())
	} else {
		return request, nil
	}
}
