package handlers

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/google/uuid"
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

	orgID := organizationBranding.OrganizationID
	if err := verifyPublicImageBelongsToOrganization(ctx, organizationBranding.LogoImageID, orgID); err != nil {
		if errors.Is(err, apierrors.ErrBadRequest) {
			http.Error(w, fmt.Sprintf("invalid logo image ID: %s", errors.Unwrap(err)), http.StatusBadRequest)
		} else {
			log.Warn("could not verify logo image", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if err := verifyPublicImageBelongsToOrganization(ctx, organizationBranding.FaviconImageID, orgID); err != nil {
		if errors.Is(err, apierrors.ErrBadRequest) {
			http.Error(w, fmt.Sprintf("invalid favicon image ID: %s", errors.Unwrap(err)), http.StatusBadRequest)
		} else {
			log.Warn("could not verify favicon image", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if err = db.UpsertOrganizationBranding(ctx, &organizationBranding); err != nil {
		log.Warn("could not save organizationBranding", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, organizationBranding)
	}
}

// verifyPublicImageBelongsToOrganization ensures the referenced branding image (logo or favicon) belongs to the
// given organization and is a public, servable image, since these images are served unauthenticated via the
// public file API. A nil imageID is valid (image not set). Validation failures are returned as ErrBadRequest.
func verifyPublicImageBelongsToOrganization(
	ctx context.Context,
	imageID *uuid.UUID,
	organizationID uuid.UUID,
) error {
	if imageID == nil {
		return nil
	}

	fileMeta, err := db.GetFileMetadataWithID(ctx, *imageID)
	if errors.Is(err, apierrors.ErrNotFound) {
		return apierrors.NewBadRequest("file does not exist")
	} else if err != nil {
		return err
	} else if fileMeta.OrganizationID == nil || *fileMeta.OrganizationID != organizationID {
		return apierrors.NewBadRequest("file does not exist")
	} else if !fileMeta.Public {
		return apierrors.NewBadRequest("file must be public")
	} else if !isServableImageContentType(fileMeta.ContentType) {
		return apierrors.NewBadRequest("file must be an image")
	}

	return nil
}
