package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func AlertConfigurationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Notifications"))

	r.Use(middleware.ProFeature, middleware.AlertsCustomerFeatureMiddleware)

	r.Get("/", respond.Handler(getAlertConfigurationsHandler)).
		With(option.Description("list all alert configurations")).
		With(option.Request(customerScopeQuery{})).
		With(option.Response(http.StatusOK, []types.AlertConfiguration{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Post("/", respond.Handler(createAlertConfigurationHandler)).
		With(option.Description("create a new alert configuration")).
		With(option.Request(types.AlertConfiguration{})).
		With(option.Response(http.StatusOK, types.AlertConfiguration{}))

	r.With(middleware.RequireReadWriteOrAdmin).
		Route("/{id}", func(r chiopenapi.Router) {
			type IDRequest struct {
				ID string `path:"id"`
			}

			r.Put("/", respond.Handler(updateAlertConfigurationHandler)).
				With(option.Description("update an existing alert configuration")).
				With(option.Request(struct {
					IDRequest
					types.AlertConfiguration
				}{})).
				With(option.Response(http.StatusOK, types.AlertConfiguration{}))

			r.Delete("/", respond.Handler(deleteAlertConfigurationHandler)).
				With(option.Description("delete an existing alert configuration")).
				With(option.Request(struct {
					IDRequest
					customerScopeQuery
				}{}))
		})
}

func getAlertConfigurationsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if customerOrgID, resp, err := customerScopeFromQuery(r); resp != nil || err != nil {
		return resp, err
	} else if configs, err := db.GetAlertConfigurations(ctx, *auth.CurrentOrgID(), customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to get alert configurations: %w", err)
	} else {
		// TODO: implement api mapping
		return respond.JSON(configs), nil
	}
}

func createAlertConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if config, resp := decodeJSON[types.AlertConfiguration](r); resp != nil {
		return resp, nil
	} else if customerOrgID, resp, err := customerScope(ctx, config.CustomerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else {
		config.OrganizationID = *auth.CurrentOrgID()
		config.CustomerOrganizationID = customerOrgID

		if err := db.CreateAlertConfiguration(ctx, &config); err != nil {
			return nil, fmt.Errorf("failed to create alert configuration: %w", err)
		} else {
			// TODO: implement api mapping
			return respond.JSON(config), nil
		}
	}
}

func updateAlertConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("id")); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if config, resp := decodeJSON[types.AlertConfiguration](r); resp != nil {
		return resp, nil
	} else if customerOrgID, resp, err := customerScope(ctx, config.CustomerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else {
		config.ID = id
		config.OrganizationID = *auth.CurrentOrgID()
		config.CustomerOrganizationID = customerOrgID

		if err := db.UpdateAlertConfiguration(ctx, &config); err != nil {
			if errors.Is(err, distrerr.ErrNotFound) {
				return alertConfigurationNotFound, nil
			} else {
				return nil, fmt.Errorf("failed to update alert configuration: %w", err)
			}
		} else {
			// TODO: implement api mapping
			return respond.JSON(config), nil
		}
	}
}

func deleteAlertConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("id")); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if customerOrgID, resp, err := customerScopeFromQuery(r); resp != nil || err != nil {
		return resp, err
	} else if err := db.DeleteAlertConfiguration(ctx, id, *auth.CurrentOrgID(), customerOrgID); err != nil {
		if errors.Is(err, distrerr.ErrNotFound) {
			return alertConfigurationNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to delete alert configuration: %w", err)
		}
	} else {
		return respond.Status(http.StatusOK), nil
	}
}
