package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func OrganizationBrandingRouter(r chiopenapi.Router) {
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", respond.Handler(getOrganizationBranding)).
		With(option.Description("Get organization branding")).
		With(option.Response(http.StatusOK, types.OrganizationBranding{}))
	r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Put("/", respond.Handler(upsertOrganizationBranding)).
		With(option.Description("Create or update organization branding")).
		With(option.Request(api.UpsertOrganizationBrandingRequest{})).
		With(option.Response(http.StatusOK, types.OrganizationBranding{}))
}

func getOrganizationBranding(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if organizationBranding, err := db.GetOrganizationBranding(
		r.Context(), *auth.CurrentOrgID(),
	); errors.Is(err, apierrors.ErrNotFound) {
		return respond.NotFound("organization branding not found"), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get organizationBranding: %w", err)
	} else {
		return respond.JSON(organizationBranding), nil
	}
}

func upsertOrganizationBranding(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if body, resp := decodeJSON[api.UpsertOrganizationBrandingRequest](r); resp != nil {
		return resp, nil
	} else {
		organizationBranding := mapping.OrganizationBrandingToInternal(body)
		organizationBranding.OrganizationID = *auth.CurrentOrgID()
		organizationBranding.UpdatedByUserAccountID = new(auth.CurrentUserID())

		orgID := organizationBranding.OrganizationID
		if err := verifyPublicImageBelongsToOrganization(
			ctx,
			organizationBranding.LogoImageID,
			orgID,
			"logo",
		); err != nil {
			if errors.Is(err, apierrors.ErrBadRequest) {
				return respond.BadRequest(err.Error()), nil
			} else {
				return nil, fmt.Errorf("could not verify logo image: %w", err)
			}
		} else if err := verifyPublicImageBelongsToOrganization(
			ctx,
			organizationBranding.FaviconImageID,
			orgID,
			"favicon",
		); err != nil {
			if errors.Is(err, apierrors.ErrBadRequest) {
				return respond.BadRequest(err.Error()), nil
			} else {
				return nil, fmt.Errorf("could not verify favicon image: %w", err)
			}
		} else if err = db.UpsertOrganizationBranding(ctx, &organizationBranding); err != nil {
			return nil, fmt.Errorf("could not save organizationBranding: %w", err)
		} else {
			return respond.JSON(organizationBranding), nil
		}
	}
}

// verifyPublicImageBelongsToOrganization ensures the referenced branding image (logo or favicon) belongs to the
// given organization and is a public, servable image, since these images are served unauthenticated via the
// public file API. A nil imageID is valid (image not set). The imageType (e.g. "logo" or "favicon") is included
// in validation messages so clients can tell which image failed. Validation failures are returned as ErrBadRequest.
func verifyPublicImageBelongsToOrganization(
	ctx context.Context,
	imageID *uuid.UUID,
	organizationID uuid.UUID,
	imageType string,
) error {
	if imageID == nil {
		return nil
	}

	fileMeta, err := db.GetFileMetadataWithID(ctx, *imageID)
	if errors.Is(err, apierrors.ErrNotFound) {
		return apierrors.NewBadRequest(fmt.Sprintf("%s file does not exist", imageType))
	} else if err != nil {
		return err
	} else if fileMeta.OrganizationID == nil || *fileMeta.OrganizationID != organizationID {
		return apierrors.NewBadRequest(fmt.Sprintf("%s file does not exist", imageType))
	} else if !fileMeta.Public {
		return apierrors.NewBadRequest(fmt.Sprintf("%s file must be public", imageType))
	} else if !isServableImageContentType(fileMeta.ContentType) {
		return apierrors.NewBadRequest(fmt.Sprintf("%s file must be an image", imageType))
	}

	return nil
}
