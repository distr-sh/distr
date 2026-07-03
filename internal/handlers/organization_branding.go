package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func OrganizationBrandingRouter(r chiopenapi.Router) {
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", getOrganizationBranding).
		With(option.Description("Get organization branding")).
		With(option.Response(http.StatusOK, types.OrganizationBranding{}))
	r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Group(func(r chiopenapi.Router) {
			r.Post("/", createOrganizationBranding).
				With(option.Description("Create organization branding")).
				With(option.Request(api.CreateOrUpdateOrganizationBrandingRequest{})).
				With(option.Response(http.StatusOK, types.OrganizationBranding{}))
			r.Put("/", updateOrganizationBranding).
				With(option.Description("Update organization branding")).
				With(option.Request(api.CreateOrUpdateOrganizationBrandingRequest{})).
				With(option.Response(http.StatusOK, types.OrganizationBranding{}))
		})
}

func getOrganizationBranding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if organizationBranding, err := db.GetOrganizationBranding(
		r.Context(), *auth.CurrentOrgID(),
	); errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if err != nil {
		internalctx.GetLogger(r.Context()).Error("failed to get organizationBranding", zap.Error(err))
		sentry.GetHubFromContext(r.Context()).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		RespondJSON(w, organizationBranding)
	}
}

func createOrganizationBranding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	organizationBranding, err := getOrganizationBrandingFromRequest(w, r)
	if err != nil {
		return
	}
	if err := setMetadataForOrganizationBranding(ctx, organizationBranding); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if err = db.CreateOrganizationBranding(r.Context(), organizationBranding); err != nil {
		log.Warn("could not create organizationBranding", zap.Error(err))
		sentry.GetHubFromContext(r.Context()).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, organizationBranding)
	}
}

func updateOrganizationBranding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	organizationBranding, err := getOrganizationBrandingFromRequest(w, r)
	if err != nil {
		return
	}
	if err := setMetadataForOrganizationBranding(ctx, organizationBranding); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if err = db.UpdateOrganizationBranding(r.Context(), organizationBranding); err != nil {
		log.Warn("could not update organizationBranding", zap.Error(err))
		sentry.GetHubFromContext(r.Context()).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, organizationBranding)
	}
}

func getOrganizationBrandingFromRequest(w http.ResponseWriter, r *http.Request) (*types.OrganizationBranding, error) {
	body, err := JsonBody[api.CreateOrUpdateOrganizationBrandingRequest](w, r)
	if err != nil {
		return nil, err
	}
	return &types.OrganizationBranding{
		Title:       body.Title,
		Description: body.Description,
		LogoImageID: body.LogoImageID,
	}, nil
}

func setMetadataForOrganizationBranding(ctx context.Context, t *types.OrganizationBranding) error {
	if auth, err := auth.Authentication.Get(ctx); err != nil {
		return err
	} else {
		t.OrganizationID = *auth.CurrentOrgID()
		t.UpdatedByUserAccountID = util.PtrTo(auth.CurrentUserID())
		t.UpdatedAt = time.Now()
		return nil
	}
}
