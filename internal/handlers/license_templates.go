package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/licensetemplate"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"github.com/stripe/stripe-go/v85"
	"go.uber.org/zap"
)

func LicenseTemplatesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Billing"))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireVendor, middleware.VendorBillingFeatureMiddleware)

	r.Get("/", getLicenseTemplates).
		With(option.Description("List all license templates")).
		With(option.Response(http.StatusOK, []types.LicenseTemplate{}))

	r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Post("/", createLicenseTemplate).
			With(option.Description("Create a new license template")).
			With(option.Request(api.CreateLicenseTemplateRequest{})).
			With(option.Response(http.StatusOK, types.LicenseTemplate{}))

		r.Route("/{licenseTemplateId}", func(r chiopenapi.Router) {
			type LicenseTemplateIDRequest struct {
				LicenseTemplateID uuid.UUID `path:"licenseTemplateId"`
			}

			r.Put("/", updateLicenseTemplate).
				With(option.Description("Update a license template")).
				With(option.Request(struct {
					LicenseTemplateIDRequest
					api.UpdateLicenseTemplateRequest
				}{})).
				With(option.Response(http.StatusOK, types.LicenseTemplate{}))

			r.Delete("/", deleteLicenseTemplate).
				With(option.Description("Delete a license template")).
				With(option.Request(LicenseTemplateIDRequest{}))
		})
	})
}

func getLicenseTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authCtx := auth.Authentication.Require(ctx)

	templates, err := db.GetLicenseTemplates(ctx, *authCtx.CurrentOrgID())
	if err != nil {
		log.Error("failed to get license templates", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, templates)
}

func createLicenseTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authCtx := auth.Authentication.Require(ctx)

	body, err := JsonBody[api.CreateLicenseTemplateRequest](w, r)
	if err != nil {
		return
	}

	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if body.PayloadTemplate == "" {
		http.Error(w, "payloadTemplate is required", http.StatusBadRequest)
		return
	}
	if body.ExpirationGracePeriodDays < 0 {
		http.Error(w, "expirationGracePeriodDays must be non-negative", http.StatusBadRequest)
		return
	}

	t := types.LicenseTemplate{
		Name:                      body.Name,
		OrganizationID:            *authCtx.CurrentOrgID(),
		PayloadTemplate:           body.PayloadTemplate,
		ExpirationGracePeriodDays: body.ExpirationGracePeriodDays,
	}

	if err := validatePayloadTemplate(t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.CreateLicenseTemplate(ctx, &t); errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "a license template with this name already exists", http.StatusBadRequest)
		return
	} else if err != nil {
		log.Error("failed to create license template", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, t)
}

func updateLicenseTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authCtx := auth.Authentication.Require(ctx)

	templateID, err := uuid.Parse(r.PathValue("licenseTemplateId"))
	if err != nil {
		http.Error(w, "licenseTemplateId is not a valid UUID", http.StatusBadRequest)
		return
	}

	body, err := JsonBody[api.UpdateLicenseTemplateRequest](w, r)
	if err != nil {
		return
	}

	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if body.PayloadTemplate == "" {
		http.Error(w, "payloadTemplate is required", http.StatusBadRequest)
		return
	}
	if body.ExpirationGracePeriodDays < 0 {
		http.Error(w, "expirationGracePeriodDays must be non-negative", http.StatusBadRequest)
		return
	}

	t := types.LicenseTemplate{
		ID:                        templateID,
		OrganizationID:            *authCtx.CurrentOrgID(),
		Name:                      body.Name,
		PayloadTemplate:           body.PayloadTemplate,
		ExpirationGracePeriodDays: body.ExpirationGracePeriodDays,
	}

	if err := validatePayloadTemplate(t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.UpdateLicenseTemplate(ctx, &t); errors.Is(err, apierrors.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "a license template with this name already exists", http.StatusBadRequest)
		return
	} else if err != nil {
		log.Error("failed to update license template", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, t)
}

func deleteLicenseTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authCtx := auth.Authentication.Require(ctx)

	templateID, err := uuid.Parse(r.PathValue("licenseTemplateId"))
	if err != nil {
		http.Error(w, "licenseTemplateId is not a valid UUID", http.StatusBadRequest)
		return
	}

	err = db.DeleteLicenseTemplateByID(ctx, templateID, *authCtx.CurrentOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		log.Error("failed to delete license template", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validatePayloadTemplate(template types.LicenseTemplate) error {
	_, err := licensetemplate.RenderPayload(template, stripe.Subscription{})
	if err != nil {
		return fmt.Errorf("%w: invalid payload template: %w", apierrors.ErrBadRequest, err)
	}

	return nil
}
