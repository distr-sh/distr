package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/controllermanifest"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ControllerImageRegistryRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Controller Image Registry"))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireAdmin)
	r.Get("/", getControllerImageRegistryHandler).
		With(option.Description("Get the image prefix the controllers of the caller's organization, or of a " +
			"customer named in the query, are pulled from")).
		With(option.Request(customerScopeQuery{})).
		With(option.Response(http.StatusOK, api.ControllerImageRegistry{}))
	r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Put("/", updateControllerImageRegistryHandler).
			With(option.Description("Set the image prefix the controller images are pulled from. The image names " +
				"docker-controller, kubernetes-controller and docker-autoheal are appended to it. An empty value " +
				"resets it to the default image registry")).
			With(option.Request(api.UpdateControllerImageRegistryRequest{})).
			With(option.Response(http.StatusOK, api.ControllerImageRegistry{}))
	})
}

func getControllerImageRegistryHandler(w http.ResponseWriter, r *http.Request) {
	customerOrgID, ok := resolveCustomerScopeFromQuery(w, r)
	if !ok {
		return
	}
	respondControllerImageRegistry(w, r, customerOrgID)
}

func updateControllerImageRegistryHandler(w http.ResponseWriter, r *http.Request) {
	request, err := JsonBody[api.UpdateControllerImageRegistryRequest](w, r)
	if err != nil {
		return
	}
	request.Normalize()
	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	customerOrgID, ok := resolveCustomerScope(w, r, request.CustomerOrganizationID)
	if !ok {
		return
	}

	var value *string
	if request.ImageRegistry != "" {
		value = &request.ImageRegistry
	}
	ctx := r.Context()
	orgID := *auth.Authentication.Require(ctx).CurrentOrgID()
	if customerOrgID != nil {
		err = db.SetCustomerOrganizationControllerImageRegistry(ctx, *customerOrgID, orgID, value)
	} else {
		err = db.SetOrganizationControllerImageRegistry(ctx, orgID, value)
	}
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if err != nil {
		respondControllerImageRegistryError(ctx, w, "failed to update controller image registry", err)
	} else {
		respondControllerImageRegistry(w, r, customerOrgID)
	}
}

func respondControllerImageRegistry(w http.ResponseWriter, r *http.Request, customerOrgID *uuid.UUID) {
	ctx := r.Context()
	org, err := db.GetOrganizationByID(ctx, *auth.Authentication.Require(ctx).CurrentOrgID())
	if err != nil {
		respondControllerImageRegistryError(ctx, w, "failed to get organization", err)
		return
	}
	if customerOrgID == nil {
		RespondJSON(w, mapping.ControllerImageRegistryToAPI(
			org.ControllerImageRegistry, controllermanifest.ImageRegistry(nil, nil)))
		return
	}
	customerOrg, err := db.GetCustomerOrganizationByID(ctx, *customerOrgID)
	if err != nil {
		respondControllerImageRegistryError(ctx, w, "failed to get customer organization", err)
		return
	}
	RespondJSON(w, mapping.ControllerImageRegistryToAPI(
		customerOrg.ControllerImageRegistry, controllermanifest.ImageRegistry(org, nil)))
}

func respondControllerImageRegistryError(ctx context.Context, w http.ResponseWriter, msg string, err error) {
	internalctx.GetLogger(ctx).Error(msg, zap.Error(err))
	sentry.GetHubFromContext(ctx).CaptureException(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
