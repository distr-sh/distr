package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

		r.Get("/revisions", getLicenseKeyRevisions).
			With(option.Description("List all revisions of a license key")).
			With(option.Request(LicenseKeyIDRequest{})).
			With(option.Response(http.StatusOK, []api.LicenseKeyRevision{}))

		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Group(func(r chiopenapi.Router) {
				r.Put("/", updateLicenseKey).
					With(option.Description("Update license key metadata and optionally create a new revision")).
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

	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	licenseKey := types.LicenseKey{
		Name:                   body.Name,
		Description:            body.Description,
		OrganizationID:         *authCtx.CurrentOrgID(),
		CustomerOrganizationID: body.CustomerOrganizationID,
		LicenseTemplateID:      body.LicenseTemplateID,
	}

	if body.LicenseTemplateID != nil {
		_, err := db.GetLicenseTemplateByID(ctx, *body.LicenseTemplateID, *authCtx.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "license template not found", http.StatusBadRequest)
			return
		} else if err != nil {
			log.Warn("could not get license template", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
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
		licenseKey.Payload = body.Payload
		licenseKey.NotBefore = &body.NotBefore
		licenseKey.ExpiresAt = &body.ExpiresAt
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

	if lk.Payload == nil {
		http.Error(w, "license key has no revision yet", http.StatusUnprocessableEntity)
		return
	}

	token, err := licensekey.GenerateToken(licensekey.FromLicenseKey(*lk), env.Host())
	if err != nil {
		log.Error("failed to generate license key token", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, "failed to generate license key token", http.StatusInternalServerError)
		return
	}
	RespondJSON(w, map[string]string{"token": token})
}

func getLicenseKeyRevisions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	lk := internalctx.GetLicenseKey(ctx)

	revisions, err := db.GetLicenseKeyRevisions(ctx, lk.ID)
	if err != nil {
		log.Error("failed to get license key revisions", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	result := make([]api.LicenseKeyRevision, len(revisions))
	for i, r := range revisions {
		t, err := licensekey.GenerateToken(licensekey.FromLicenseKeyAndRevision(*lk, r), env.Host())
		if err != nil {
			log.Error("failed to generate license key token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, "failed to generate license key token", http.StatusInternalServerError)
			return
		}
		result[i] = api.LicenseKeyRevision{LicenseKeyRevision: r, Token: t}
	}

	RespondJSON(w, result)
}

func updateLicenseKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	existing := internalctx.GetLicenseKey(ctx)

	body, err := JsonBody[api.UpdateLicenseKeyRequest](w, r)
	if err != nil {
		return
	}

	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	authCtx := auth.Authentication.Require(ctx)

	if body.LicenseTemplateID != nil {
		_, err := db.GetLicenseTemplateByID(ctx, *body.LicenseTemplateID, *authCtx.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "license template not found", http.StatusBadRequest)
			return
		} else if err != nil {
			log.Warn("could not get license template", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if body.Payload != nil {
		if err := licensekey.ValidatePayload(*body.Payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	effectiveNotBefore := body.NotBefore
	if effectiveNotBefore == nil {
		effectiveNotBefore = existing.NotBefore
	}
	effectiveExpiresAt := body.ExpiresAt
	if effectiveExpiresAt == nil {
		effectiveExpiresAt = existing.ExpiresAt
	}
	if effectiveExpiresAt != nil && effectiveNotBefore != nil && !effectiveExpiresAt.After(*effectiveNotBefore) {
		http.Error(w, "expiresAt must be after notBefore", http.StatusBadRequest)
		return
	}

	needsRevision, err := licenseKeyRevisionChanged(existing, body)
	if err != nil {
		log.Warn("could not compare license key revision", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result *types.LicenseKey
	var errorHandled bool
	err = db.RunTx(ctx, func(ctx context.Context) error {
		if r, err := db.UpdateLicenseKeyMetadata(
			ctx, existing.ID, body.Name, body.Description, body.LicenseTemplateID,
		); errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "A license key with this name already exists", http.StatusBadRequest)
			errorHandled = true
			return err
		} else if err != nil {
			log.Warn("could not update license key", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			errorHandled = true
			return err
		} else {
			result = r
		}

		if needsRevision {
			notBefore := body.NotBefore
			if notBefore == nil {
				notBefore = existing.NotBefore
			}
			expiresAt := body.ExpiresAt
			if expiresAt == nil {
				expiresAt = existing.ExpiresAt
			}
			var payload json.RawMessage
			if body.Payload != nil {
				payload = *body.Payload
			} else {
				payload = existing.Payload
			}

			if notBefore == nil || expiresAt == nil || payload == nil {
				http.Error(w, "notBefore, expiresAt and payload are required to create a revision", http.StatusBadRequest)
				errorHandled = true
				return errors.New("missing revision fields")
			}

			revision := types.LicenseKeyRevision{
				LicenseKeyID: existing.ID,
				NotBefore:    *notBefore,
				ExpiresAt:    *expiresAt,
				Payload:      payload,
			}

			if err := db.CreateLicenseKeyRevision(ctx, &revision); err != nil {
				log.Warn("could not create license key revision", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				errorHandled = true
				return err
			}
			result.NotBefore = &revision.NotBefore
			result.ExpiresAt = &revision.ExpiresAt
			result.Payload = revision.Payload
			result.LastRevisedAt = &revision.CreatedAt
		}

		return nil
	})

	if err != nil {
		if !errorHandled {
			log.Warn("update license key error", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

// licenseKeyRevisionChanged returns true if any of the revision fields differ
// between the existing (latest) revision and the incoming request.
func licenseKeyRevisionChanged(existing *types.LicenseKey, body api.UpdateLicenseKeyRequest) (bool, error) {
	if body.NotBefore != nil {
		if existing.NotBefore == nil || !existing.NotBefore.Equal(body.NotBefore.UTC().Truncate(time.Microsecond)) {
			return true, nil
		}
	}

	if body.ExpiresAt != nil {
		if existing.ExpiresAt == nil || !existing.ExpiresAt.Equal(body.ExpiresAt.UTC().Truncate(time.Microsecond)) {
			return true, nil
		}
	}

	if body.Payload != nil {
		if existing.Payload == nil {
			return true, nil
		}
		equal, err := payloadsEqual(existing.Payload, *body.Payload)
		if err != nil {
			return false, err
		}
		return !equal, nil
	}

	return false, nil
}

func payloadsEqual(a, b json.RawMessage) (bool, error) {
	na, err := normalizeJSON(a)
	if err != nil {
		return false, err
	}
	nb, err := normalizeJSON(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(na, nb), nil
}

func normalizeJSON(raw json.RawMessage) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
