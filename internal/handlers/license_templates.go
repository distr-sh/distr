package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/licensetemplate"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"github.com/stripe/stripe-go/v86"
)

func LicenseTemplatesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Billing"))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireVendor, middleware.VendorBillingFeatureMiddleware)

	r.Get("/", respond.Handler(getLicenseTemplates)).
		With(option.Description("List all license templates")).
		With(option.Response(http.StatusOK, []types.LicenseTemplate{}))

	r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Post("/", respond.Handler(createLicenseTemplate)).
			With(option.Description("Create a new license template")).
			With(option.Request(api.CreateLicenseTemplateRequest{})).
			With(option.Response(http.StatusOK, types.LicenseTemplate{}))

		r.Route("/{licenseTemplateId}", func(r chiopenapi.Router) {
			type LicenseTemplateIDRequest struct {
				LicenseTemplateID uuid.UUID `path:"licenseTemplateId"`
			}

			r.Put("/", respond.Handler(updateLicenseTemplate)).
				With(option.Description("Update a license template")).
				With(option.Request(struct {
					LicenseTemplateIDRequest
					api.UpdateLicenseTemplateRequest
				}{})).
				With(option.Response(http.StatusOK, types.LicenseTemplate{}))

			r.Delete("/", respond.Handler(deleteLicenseTemplate)).
				With(option.Description("Delete a license template")).
				With(option.Request(LicenseTemplateIDRequest{}))
		})
	})
}

func getLicenseTemplates(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)

	if templates, err := db.GetLicenseTemplates(ctx, *authCtx.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get license templates: %w", err)
	} else {
		return respond.JSON(templates), nil
	}
}

func createLicenseTemplate(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)

	if body, resp := decodeJSON[api.CreateLicenseTemplateRequest](r); resp != nil {
		return resp, nil
	} else if body.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else if body.PayloadTemplate == "" {
		return respond.BadRequest("payloadTemplate is required"), nil
	} else if body.ExpirationGracePeriodDays < 0 {
		return respond.BadRequest("expirationGracePeriodDays must be non-negative"), nil
	} else {
		t := types.LicenseTemplate{
			Name:                      body.Name,
			OrganizationID:            *authCtx.CurrentOrgID(),
			PayloadTemplate:           body.PayloadTemplate,
			ExpirationGracePeriodDays: body.ExpirationGracePeriodDays,
		}

		if err := validatePayloadTemplate(t); err != nil {
			return respond.BadRequest(err.Error()), nil
		} else if err := db.CreateLicenseTemplate(ctx, &t); errors.Is(err, apierrors.ErrConflict) {
			return respond.BadRequest("a license template with this name already exists"), nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to create license template: %w", err)
		} else {
			return respond.JSON(t), nil
		}
	}
}

var licenseTemplateNotFound = respond.NotFound("license template not found")

func updateLicenseTemplate(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)

	if templateID, err := uuid.Parse(r.PathValue("licenseTemplateId")); err != nil {
		return respond.BadRequest("licenseTemplateId is not a valid UUID"), nil
	} else if body, resp := decodeJSON[api.UpdateLicenseTemplateRequest](r); resp != nil {
		return resp, nil
	} else if body.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else if body.PayloadTemplate == "" {
		return respond.BadRequest("payloadTemplate is required"), nil
	} else if body.ExpirationGracePeriodDays < 0 {
		return respond.BadRequest("expirationGracePeriodDays must be non-negative"), nil
	} else {
		t := types.LicenseTemplate{
			ID:                        templateID,
			OrganizationID:            *authCtx.CurrentOrgID(),
			Name:                      body.Name,
			PayloadTemplate:           body.PayloadTemplate,
			ExpirationGracePeriodDays: body.ExpirationGracePeriodDays,
		}

		if err := validatePayloadTemplate(t); err != nil {
			return respond.BadRequest(err.Error()), nil
		} else if err := db.UpdateLicenseTemplate(ctx, &t); errors.Is(err, apierrors.ErrNotFound) {
			return licenseTemplateNotFound, nil
		} else if errors.Is(err, apierrors.ErrConflict) {
			return respond.BadRequest("a license template with this name already exists"), nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to update license template: %w", err)
		} else {
			return respond.JSON(t), nil
		}
	}
}

func deleteLicenseTemplate(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)

	if templateID, err := uuid.Parse(r.PathValue("licenseTemplateId")); err != nil {
		return respond.BadRequest("licenseTemplateId is not a valid UUID"), nil
	} else if err := db.DeleteLicenseTemplateByID(
		ctx, templateID, *authCtx.CurrentOrgID(),
	); errors.Is(err, apierrors.ErrNotFound) {
		return licenseTemplateNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to delete license template: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func validatePayloadTemplate(template types.LicenseTemplate) error {
	_, err := licensetemplate.RenderPayload(template, stripe.Subscription{})
	if err != nil {
		return fmt.Errorf("%w: invalid payload template: %w", apierrors.ErrBadRequest, err)
	}

	return nil
}
