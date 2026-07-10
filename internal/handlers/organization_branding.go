package handlers

import (
	"context"
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
		Put("/", upsertOrganizationBranding).
		With(option.Description("Create or update organization branding")).
		With(option.Request(api.UpsertOrganizationBrandingRequest{})).
		With(option.Response(http.StatusOK, types.OrganizationBranding{}))
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

func upsertOrganizationBranding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	body, err := JsonBody[api.UpsertOrganizationBrandingRequest](w, r)
	if err != nil {
		return
	}

	organizationBranding := mapping.OrganizationBrandingToInternal(body)
	organizationBranding.OrganizationID = *auth.CurrentOrgID()
	organizationBranding.UpdatedByUserAccountID = util.PtrTo(auth.CurrentUserID())

	if err := verifyLogoImageBelongsToOrganization(ctx, organizationBranding); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if err := verifyFaviconImageBelongsToOrganization(ctx, organizationBranding); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if err = db.UpsertOrganizationBranding(ctx, &organizationBranding); err != nil {
		log.Warn("could not save organizationBranding", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, organizationBranding)
	}
}

// verifyLogoImageBelongsToOrganization ensures the referenced logo image belongs to the current
// organization. Without this check a user could reference an arbitrary File (e.g. from another
// organization), whose bytes are then embedded into outbound e-mail content.
func verifyLogoImageBelongsToOrganization(ctx context.Context, t types.OrganizationBranding) error {
	if t.LogoImageID == nil {
		return nil
	}
	file, err := db.GetFileWithID(ctx, *t.LogoImageID)
	if errors.Is(err, apierrors.ErrNotFound) {
		return errors.New("logo image does not exist")
	} else if err != nil {
		return err
	} else if file.OrganizationID == nil || *file.OrganizationID != t.OrganizationID {
		return errors.New("logo image does not belong to the organization")
	}
	return nil
}

// verifyFaviconImageBelongsToOrganization ensures the referenced favicon image belongs to the current
// organization and is public. The favicon is served without authentication via the public file API (the
// browser loads it as a plain resource), so it must be public and must not leak another organization's asset.
func verifyFaviconImageBelongsToOrganization(ctx context.Context, t types.OrganizationBranding) error {
	if t.FaviconImageID == nil {
		return nil
	}
	file, err := db.GetFileMetadataWithID(ctx, *t.FaviconImageID)
	if errors.Is(err, apierrors.ErrNotFound) {
		return errors.New("favicon image does not exist")
	} else if err != nil {
		return err
	} else if file.OrganizationID == nil || *file.OrganizationID != t.OrganizationID {
		return errors.New("favicon image does not belong to the organization")
	} else if !file.Public {
		return errors.New("favicon image must be public")
	}
	return nil
}
