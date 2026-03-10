package handlers

import (
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/licensekey"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func LicenseKeysRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Licensing"))
	r.Use(middleware.RequireOrgAndRole, middleware.LicensingFeatureFlagEnabledMiddleware)

	r.Get("/", getLicenseKeys).
		With(option.Description("List all license keys")).
		With(option.Response(http.StatusOK, []types.LicenseKey{}))

	r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", createLicenseKey).
		With(option.Description("Create a new license key")).
		With(option.Request(api.CreateLicenseKeyRequest{})).
		With(option.Response(http.StatusOK, types.LicenseKey{}))

	r.With(licenseKeyMiddleware).Route("/{licenseKeyId}", func(r chiopenapi.Router) {
		type LicenseKeyIDRequest struct {
			LicenseKeyID uuid.UUID `path:"licenseKeyId"`
		}

		r.Get("/token", getLicenseKeyToken).
			With(option.Description("Generate and retrieve the license key token")).
			With(option.Request(LicenseKeyIDRequest{})).
			With(option.Response(http.StatusOK, map[string]string{}))

		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Group(func(r chiopenapi.Router) {
				r.Put("/", updateLicenseKey).
					With(option.Description("Update license key name and description")).
					With(option.Request(struct {
						LicenseKeyIDRequest
						api.UpdateLicenseKeyRequest
					}{})).
					With(option.Response(http.StatusOK, types.LicenseKey{}))
				r.Delete("/", deleteLicenseKey).
					With(option.Description("Delete a license key")).
					With(option.Request(LicenseKeyIDRequest{}))
			})
	})
}

func getLicenseKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	if auth.CurrentCustomerOrgID() == nil {
		if licenseKeys, err := db.GetLicenseKeys(ctx, *auth.CurrentOrgID()); err != nil {
			log.Error("failed to get license keys", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			RespondJSON(w, licenseKeys)
		}
	} else {
		if licenseKeys, err := db.GetLicenseKeysByCustomerOrgID(
			ctx, *auth.CurrentCustomerOrgID(), *auth.CurrentOrgID(),
		); err != nil {
			log.Error("failed to get license keys", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else {
			RespondJSON(w, licenseKeys)
		}
	}
}

func createLicenseKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authCtx := auth.Authentication.Require(ctx)

	body, err := JsonBody[api.CreateLicenseKeyRequest](w, r)
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

	if err := licensekey.ValidatePayload(body.Payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	licenseKey := types.LicenseKey{
		Name:                   body.Name,
		Description:            body.Description,
		Payload:                body.Payload,
		NotBefore:              body.NotBefore,
		ExpiresAt:              body.ExpiresAt,
		OrganizationID:         *authCtx.CurrentOrgID(),
		CustomerOrganizationID: body.CustomerOrganizationID,
	}

	if err := db.CreateLicenseKey(ctx, &licenseKey); errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "A license key with this name already exists", http.StatusBadRequest)
		return
	} else if err != nil {
		log.Warn("could not create license key", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, licenseKey)
}

func getLicenseKeyToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	lk := internalctx.GetLicenseKey(ctx)

	token, err := licensekey.GenerateToken(lk, env.Host())
	if err != nil {
		log.Error("failed to generate license key token", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, "failed to generate license key token", http.StatusInternalServerError)
		return
	}
	RespondJSON(w, map[string]string{"token": token})
}

func updateLicenseKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	existing := internalctx.GetLicenseKey(ctx)

	body, err := JsonBody[api.UpdateLicenseKeyRequest](w, r)
	if err != nil {
		return
	}

	result, err := db.UpdateLicenseKeyMetadata(ctx, existing.ID, body.Name, body.Description)
	if errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "A license key with this name already exists", http.StatusBadRequest)
	} else if err != nil {
		log.Warn("could not update license key", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, result)
	}
}

func deleteLicenseKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	licenseKey := internalctx.GetLicenseKey(ctx)

	if err := db.DeleteLicenseKeyWithID(ctx, licenseKey.ID); err != nil {
		log.Warn("error deleting license key", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func licenseKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := auth.Authentication.Require(ctx)
		if licenseKeyID, err := uuid.Parse(r.PathValue("licenseKeyId")); err != nil {
			http.Error(w, "licenseKeyId is not a valid UUID", http.StatusBadRequest)
		} else if licenseKey, err := db.GetLicenseKeyByID(ctx, licenseKeyID); errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get license key", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else if licenseKey.OrganizationID != *authCtx.CurrentOrgID() {
			w.WriteHeader(http.StatusNotFound)
		} else if authCtx.CurrentCustomerOrgID() != nil &&
			(licenseKey.CustomerOrganizationID == nil || *licenseKey.CustomerOrganizationID != *authCtx.CurrentCustomerOrgID()) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			ctx = internalctx.WithLicenseKey(ctx, licenseKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}
