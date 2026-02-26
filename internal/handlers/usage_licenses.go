package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/usagelicense"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

type CreateUsageLicenseRequest struct {
	Name                   string          `json:"name"`
	Description            *string         `json:"description,omitempty"`
	Payload                json.RawMessage `json:"payload"`
	NotBefore              time.Time       `json:"notBefore"`
	ExpiresAt              time.Time       `json:"expiresAt"`
	CustomerOrganizationID *uuid.UUID      `json:"customerOrganizationId,omitempty"`
}

type UpdateUsageLicenseRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

func UsageLicensesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Licensing"))
	r.Use(middleware.RequireOrgAndRole, middleware.LicensingFeatureFlagEnabledMiddleware)

	r.Get("/", getUsageLicenses).
		With(option.Description("List all usage licenses")).
		With(option.Response(http.StatusOK, []types.UsageLicense{}))

	r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Group(func(r chiopenapi.Router) {
			r.Post("/", createUsageLicense).
				With(option.Description("Create a new usage license")).
				With(option.Request(CreateUsageLicenseRequest{})).
				With(option.Response(http.StatusOK, types.UsageLicense{}))

			r.With(usageLicenseMiddleware).Route("/{usageLicenseId}", func(r chiopenapi.Router) {
				type UsageLicenseIDRequest struct {
					UsageLicenseID uuid.UUID `path:"usageLicenseId"`
				}

				r.Put("/", updateUsageLicense).
					With(option.Description("Update usage license name and description")).
					With(option.Request(struct {
						UsageLicenseIDRequest
						UpdateUsageLicenseRequest
					}{})).
					With(option.Response(http.StatusOK, types.UsageLicense{}))
				r.Delete("/", deleteUsageLicense).
					With(option.Description("Delete a usage license")).
					With(option.Request(UsageLicenseIDRequest{}))
			})
		})
}

func getUsageLicenses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	if auth.CurrentCustomerOrgID() == nil {
		if licenses, err := db.GetUsageLicenses(ctx, *auth.CurrentOrgID()); err != nil {
			log.Error("failed to get usage licenses", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			RespondJSON(w, licenses)
		}
	} else {
		if licenses, err := db.GetUsageLicensesByCustomerOrgID(
			ctx, *auth.CurrentCustomerOrgID(), *auth.CurrentOrgID(),
		); err != nil {
			log.Error("failed to get usage licenses", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			RespondJSON(w, licenses)
		}
	}
}

func createUsageLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authCtx := auth.Authentication.Require(ctx)

	body, err := JsonBody[CreateUsageLicenseRequest](w, r)
	if err != nil {
		return
	}

	if body.NotBefore.IsZero() {
		http.Error(w, "notBefore is required", http.StatusBadRequest)
		return
	}
	if body.ExpiresAt.IsZero() {
		http.Error(w, "expiresAt is required", http.StatusBadRequest)
		return
	}
	if !body.ExpiresAt.After(body.NotBefore) {
		http.Error(w, "expiresAt must be after notBefore", http.StatusBadRequest)
		return
	}

	if err := usagelicense.ValidatePayload(body.Payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	license := types.UsageLicense{
		Name:                   body.Name,
		Description:            body.Description,
		Payload:                body.Payload,
		NotBefore:              body.NotBefore,
		ExpiresAt:              body.ExpiresAt,
		OrganizationID:         *authCtx.CurrentOrgID(),
		CustomerOrganizationID: body.CustomerOrganizationID,
	}

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.CreateUsageLicense(ctx, &license); errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "A usage license with this name already exists", http.StatusBadRequest)
			return err
		} else if err != nil {
			log.Warn("could not create usage license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		token, err := usagelicense.GenerateToken(&license, env.Host())
		if err != nil {
			log.Error("failed to generate usage license token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, "failed to generate license token", http.StatusInternalServerError)
			return err
		}

		if err := db.UpdateUsageLicenseToken(ctx, license.ID, token); err != nil {
			log.Error("failed to update usage license token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		license.Token = token

		RespondJSON(w, license)
		return nil
	})
}

func updateUsageLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	existing := internalctx.GetUsageLicense(ctx)

	body, err := JsonBody[UpdateUsageLicenseRequest](w, r)
	if err != nil {
		return
	}

	result, err := db.UpdateUsageLicenseMetadata(ctx, existing.ID, body.Name, body.Description)
	if errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "A usage license with this name already exists", http.StatusBadRequest)
	} else if err != nil {
		log.Warn("could not update usage license", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, result)
	}
}

func deleteUsageLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	license := internalctx.GetUsageLicense(ctx)

	if err := db.DeleteUsageLicenseWithID(ctx, license.ID); err != nil {
		log.Warn("error deleting usage license", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func usageLicenseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := auth.Authentication.Require(ctx)
		if licenseID, err := uuid.Parse(r.PathValue("usageLicenseId")); err != nil {
			http.Error(w, "usageLicenseId is not a valid UUID", http.StatusBadRequest)
		} else if license, err := db.GetUsageLicenseByID(ctx, licenseID); errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get usage license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else if license.OrganizationID != *authCtx.CurrentOrgID() {
			w.WriteHeader(http.StatusNotFound)
		} else {
			ctx = internalctx.WithUsageLicense(ctx, license)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}
