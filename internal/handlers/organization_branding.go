package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

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
				With(option.Request(nil, option.ContentType("multipart/formdata"))).
				With(option.Response(http.StatusOK, types.OrganizationBranding{}))
			r.Put("/", updateOrganizationBranding).
				With(option.Description("Update organization branding")).
				With(option.Request(nil, option.ContentType("multipart/formdata"))).
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

	if organizationBranding, err := getOrganizationBrandingFromRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if err := setMetadataForOrganizationBranding(ctx, organizationBranding); err != nil {
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

	if organizationBranding, err := getOrganizationBrandingFromRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if err := setMetadataForOrganizationBranding(ctx, organizationBranding); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if err = db.UpdateOrganizationBranding(r.Context(), organizationBranding); err != nil {
		log.Warn("could not create organizationBranding", zap.Error(err))
		sentry.GetHubFromContext(r.Context()).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, organizationBranding)
	}
}

func getOrganizationBrandingFromRequest(r *http.Request) (*types.OrganizationBranding, error) {
	if err := r.ParseMultipartForm(102400); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}
	organizationBranding := types.OrganizationBranding{
		Title:       util.PtrTo(r.Form.Get("title")),
		Description: util.PtrTo(r.Form.Get("description")),
	}

	if file, head, err := r.FormFile("logo"); err != nil {
		if !errors.Is(err, http.ErrMissingFile) {
			return nil, err
		} else {
			// no logo uploaded
			organizationBranding.Logo = nil
			organizationBranding.LogoFileName = nil
			organizationBranding.LogoContentType = nil
		}
	} else if head.Size > 102400 {
		return nil, errors.New("file too large (max 100 KiB)")
	} else if data, err := io.ReadAll(file); err != nil {
		return nil, err
	} else {
		organizationBranding.Logo = data
		organizationBranding.LogoFileName = &head.Filename
		organizationBranding.LogoContentType = util.PtrTo(head.Header.Get("Content-Type"))
	}

	return &organizationBranding, nil
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
