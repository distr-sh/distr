package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/controllermanifest"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func CustomerSettingsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Customers"))
	r.Use(
		middleware.RequireOrgAndRole,
		middleware.RequireAdmin,
		middleware.CustomerSettingsFeatureMiddleware,
		middleware.SettingsCustomerFeatureMiddleware,
	)

	r.Get("/", respond.Handler(getCustomerSettingsHandler)).
		With(option.Description("Get the settings of a customer organization")).
		With(option.Request(customerScopeQuery{})).
		With(option.Response(http.StatusOK, api.CustomerSettingsResponse{}))

	r.With(middleware.BlockSuperAdmin).
		Put("/", respond.Handler(updateCustomerSettingsHandler)).
		With(option.Description("Update the settings of a customer organization")).
		With(option.Request(struct {
			customerScopeQuery
			api.UpdateCustomerSettingsRequest
		}{})).
		With(option.Response(http.StatusOK, api.CustomerSettingsResponse{}))
}

var customerSettingsNotAllowed = respond.Forbidden("settings are not enabled for this customer")

func getCustomerSettingsHandler(r *http.Request) (respond.Responder, error) {
	if customerOrg, resp, err := requireCustomerSettings(r); resp != nil || err != nil {
		return resp, err
	} else {
		return customerSettingsResponse(r.Context(), *customerOrg)
	}
}

func updateCustomerSettingsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()

	body, resp := decodeJSON[api.UpdateCustomerSettingsRequest](r)
	if resp != nil {
		return resp, nil
	}
	body.Normalize()
	if customerOrg, resp, err := requireCustomerSettings(r); resp != nil || err != nil {
		return resp, err
	} else if err := body.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if updated, err := db.UpdateCustomerOrganizationControllerImageRepository(
		ctx, customerOrg.ID, *auth.Authentication.Require(ctx).CurrentOrgID(), body.ControllerImageRepository,
	); errors.Is(err, distrerr.ErrNotFound) {
		return customerOrganizationNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to update customer settings: %w", err)
	} else {
		return customerSettingsResponse(ctx, *updated)
	}
}

// requireCustomerSettings loads the customer organization a request is scoped to and rejects it unless
// that customer has the settings feature. SettingsCustomerFeatureMiddleware only covers customer users,
// vendors and partners name the customer in the query.
func requireCustomerSettings(r *http.Request) (*types.CustomerOrganization, respond.Responder, error) {
	if customerOrgID, resp, err := customerScopeFromQuery(r); resp != nil || err != nil {
		return nil, resp, err
	} else if customerOrgID == nil {
		return nil, respond.BadRequest("customer organization ID is required"), nil
	} else if customerOrg, err := db.GetCustomerOrganizationByID(r.Context(), *customerOrgID); errors.Is(
		err, distrerr.ErrNotFound,
	) {
		return nil, invalidCustomerScope, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get customer organization: %w", err)
	} else if !customerOrg.HasFeature(types.CustomerOrganizationFeatureSettings) {
		return nil, customerSettingsNotAllowed, nil
	} else {
		return &customerOrg.CustomerOrganization, nil, nil
	}
}

func customerSettingsResponse(
	ctx context.Context,
	customerOrg types.CustomerOrganization,
) (respond.Responder, error) {
	if org, err := db.GetOrganizationWithBranding(ctx, *auth.Authentication.Require(ctx).CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	} else {
		defaultRepository := controllermanifest.OrganizationImageRepository(ctx, *org)
		return respond.JSON(mapping.CustomerSettingsToAPI(customerOrg, defaultRepository)), nil
	}
}
