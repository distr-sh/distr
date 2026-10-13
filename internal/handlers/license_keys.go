package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func LicenseKeysRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Licensing"))
	r.Use(middleware.RequireOrgAndRole, middleware.LicensingFeatureFlagEnabledMiddleware)

	r.Get("/", respond.Handler(getLicenseKeys)).
		With(option.Description("List all license keys")).
		With(option.Response(http.StatusOK, []types.LicenseKey{}))

	r.With(middleware.RequireVendorOrPartner, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", respond.Handler(createLicenseKey)).
		With(option.Description("Create a new license key")).
		With(option.Request(api.CreateLicenseKeyRequest{})).
		With(option.Response(http.StatusOK, types.LicenseKey{}))

	r.With(licenseKeyMiddleware).Route("/{licenseKeyId}", func(r chiopenapi.Router) {
		type LicenseKeyIDRequest struct {
			LicenseKeyID uuid.UUID `path:"licenseKeyId"`
		}

		r.Get("/token", respond.Handler(getLicenseKeyToken)).
			With(option.Description("Generate and retrieve the license key token")).
			With(option.Request(LicenseKeyIDRequest{})).
			With(option.Response(http.StatusOK, map[string]string{}))

		r.Get("/revisions", respond.Handler(getLicenseKeyRevisions)).
			With(option.Description("List all revisions of a license key")).
			With(option.Request(LicenseKeyIDRequest{})).
			With(option.Response(http.StatusOK, []api.LicenseKeyRevision{}))

		r.With(middleware.RequireVendorOrPartner, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Group(func(r chiopenapi.Router) {
				r.Put("/", updateLicenseKey).
					With(option.Description("Update license key metadata and optionally create a new revision")).
					With(option.Request(struct {
						LicenseKeyIDRequest
						api.UpdateLicenseKeyRequest
					}{})).
					With(option.Response(http.StatusOK, api.UpdateLicenseKeyResponse{})).
					With(option.Response(http.StatusConflict, api.AffectedDeploymentsConflictResponse{}))
				r.Delete("/", respond.Handler(deleteLicenseKey)).
					With(option.Description("Delete a license key")).
					With(option.Request(LicenseKeyIDRequest{})).
					With(option.Response(http.StatusConflict, api.AffectedDeploymentsConflictResponse{}))
			})
	})
}

func getLicenseKeys(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)

	var (
		licenseKeys []types.LicenseKey
		err         error
	)
	if authCtx.CurrentCustomerOrgID() != nil {
		licenseKeys, err = db.GetLicenseKeysByCustomerOrgID(ctx, *authCtx.CurrentCustomerOrgID(), *authCtx.CurrentOrgID())
	} else if partnerOrgID := authCtx.CurrentPartnerOrgID(); partnerOrgID != nil {
		licenseKeys, err = db.GetLicenseKeysByPartnerOrgID(ctx, *partnerOrgID, *authCtx.CurrentOrgID())
	} else {
		licenseKeys, err = db.GetLicenseKeys(ctx, *authCtx.CurrentOrgID())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get license keys: %w", err)
	}
	return respond.JSON(licenseKeys), nil
}

func createLicenseKey(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)

	if body, resp := decodeJSON[api.CreateLicenseKeyRequest](r); resp != nil {
		return resp, nil
	} else if body.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else {
		licenseKey := types.LicenseKey{
			Name:                   body.Name,
			Description:            body.Description,
			OrganizationID:         *authCtx.CurrentOrgID(),
			CustomerOrganizationID: body.CustomerOrganizationID,
			LicenseTemplateID:      body.LicenseTemplateID,
		}

		if body.LicenseTemplateID != nil {
			if _, err := db.GetLicenseTemplateByID(
				ctx, *body.LicenseTemplateID, *authCtx.CurrentOrgID(),
			); errors.Is(err, apierrors.ErrNotFound) {
				return respond.BadRequest("license template not found"), nil
			} else if err != nil {
				return nil, fmt.Errorf("could not get license template: %w", err)
			}
		} else if body.NotBefore.IsZero() {
			return respond.BadRequest("notBefore is required"), nil
		} else if body.ExpiresAt.IsZero() {
			return respond.BadRequest("expiresAt is required"), nil
		} else if !body.ExpiresAt.After(body.NotBefore) {
			return respond.BadRequest("expiresAt must be after notBefore"), nil
		} else if err := licensekey.ValidatePayload(body.Payload); err != nil {
			return respond.BadRequest(err.Error()), nil
		} else {
			licenseKey.Payload = body.Payload
			licenseKey.NotBefore = &body.NotBefore
			licenseKey.ExpiresAt = &body.ExpiresAt
		}

		if partnerOrgID := authCtx.CurrentPartnerOrgID(); partnerOrgID != nil {
			if body.CustomerOrganizationID == nil {
				return respond.BadRequest("customer organization is required"), nil
			} else if co, coErr := db.GetCustomerOrganizationByID(ctx, *body.CustomerOrganizationID); coErr != nil ||
				!util.PtrEq(co.PartnerOrganizationID, partnerOrgID) {
				return respond.Forbidden("customer is not assigned to your partner organization"), nil
			}
		}

		if err := db.CreateLicenseKey(ctx, &licenseKey); errors.Is(err, apierrors.ErrConflict) {
			return respond.BadRequest("A license key with this name already exists"), nil
		} else if err != nil {
			return nil, fmt.Errorf("could not create license key: %w", err)
		} else {
			return respond.JSON(licenseKey), nil
		}
	}
}

func getLicenseKeyToken(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	lk := internalctx.GetLicenseKey(ctx)

	if lk.Payload == nil {
		return respond.Error(http.StatusUnprocessableEntity, "license key has no revision yet"), nil
	} else if token, err := licensekey.GenerateToken(licensekey.FromLicenseKey(*lk), env.Host()); err != nil {
		return nil, fmt.Errorf("failed to generate license key token: %w", err)
	} else {
		return respond.JSON(map[string]string{"token": token}), nil
	}
}

func getLicenseKeyRevisions(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	lk := internalctx.GetLicenseKey(ctx)

	if revisions, err := db.GetLicenseKeyRevisions(ctx, lk.ID); err != nil {
		return nil, fmt.Errorf("failed to get license key revisions: %w", err)
	} else {
		result := make([]api.LicenseKeyRevision, len(revisions))
		for i, r := range revisions {
			if t, err := licensekey.GenerateToken(licensekey.FromLicenseKeyAndRevision(*lk, r), env.Host()); err != nil {
				return nil, fmt.Errorf("failed to generate license key token: %w", err)
			} else {
				result[i] = api.LicenseKeyRevision{LicenseKeyRevision: r, Token: t}
			}
		}
		return respond.JSON(result), nil
	}
}

func updateLicenseKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	existing := internalctx.GetLicenseKey(ctx)

	body, err := JsonBody[api.UpdateLicenseKeyRequest](w, r)
	if err != nil {
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

	affected := []api.AffectedDeployment{}
	if needsRevision && existing.CustomerOrganizationID != nil {
		hypotheticalLicenseKey, err := updatedLicenseKeyForRequest(existing, body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		affected, err = findAffectedDeploymentsByLicenseKey(
			ctx,
			*authCtx.CurrentOrgID(),
			*existing.CustomerOrganizationID,
			hypotheticalLicenseKey,
		)
		if err != nil {
			_ = deploymentValuesError(ctx, w, err, "invalid deployment values")
			return
		}
	}

	confirm := r.URL.Query().Get("confirm") == "true"
	if !confirm && len(affected) > 0 {
		RespondJSONWithStatus(w, http.StatusConflict,
			api.AffectedDeploymentsConflictResponse{AffectedDeployments: affected})
		return
	}

	var result *types.LicenseKey
	var errorHandled bool
	err = db.RunTx(ctx, func(ctx context.Context) error {
		if r, err := db.UpdateLicenseKeyMetadata(
			ctx, existing.ID, body.Description, body.LicenseTemplateID,
		); err != nil {
			log.Warn("could not update license key", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			errorHandled = true
			return err
		} else {
			result = r
		}

		if needsRevision {
			var handled bool
			if handled, err = applyLicenseKeyRevision(ctx, w, log, existing, body, result); err != nil {
				errorHandled = handled
				return err
			}
		}

		return triggerAffectedDeployments(
			ctx, affected, new(authCtx.CurrentUserID()), types.DeploymentRevisionTriggerLicenseKeyChange,
		)
	})

	if err != nil {
		if !errorHandled {
			log.Warn("update license key error", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		RespondJSON(w, api.UpdateLicenseKeyResponse{
			LicenseKey:          *result,
			AffectedDeployments: affected,
		})
	}
}

func applyLicenseKeyRevision(
	ctx context.Context,
	w http.ResponseWriter,
	log *zap.Logger,
	existing *types.LicenseKey,
	body api.UpdateLicenseKeyRequest,
	result *types.LicenseKey,
) (errorHandled bool, err error) {
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
		return true, errors.New("missing revision fields")
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
		return true, err
	}
	result.NotBefore = &revision.NotBefore
	result.ExpiresAt = &revision.ExpiresAt
	result.Payload = revision.Payload
	result.LastRevisedAt = &revision.CreatedAt
	return false, nil
}

func updatedLicenseKeyForRequest(
	existing *types.LicenseKey,
	body api.UpdateLicenseKeyRequest,
) (types.LicenseKey, error) {
	result := *existing
	result.Description = body.Description
	result.LicenseTemplateID = body.LicenseTemplateID

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
		return types.LicenseKey{}, errors.New("notBefore, expiresAt and payload are required to create a revision")
	}

	revisedAt := time.Now().UTC().Truncate(time.Microsecond)
	result.NotBefore = notBefore
	result.ExpiresAt = expiresAt
	result.Payload = payload
	result.LastRevisedAt = &revisedAt
	return result, nil
}

func deleteLicenseKey(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)
	licenseKey := internalctx.GetLicenseKey(ctx)

	if licenseKey.CustomerOrganizationID != nil {
		if referencing, err := findDeploymentsReferencingLicenseKey(
			ctx, *authCtx.CurrentOrgID(), *licenseKey.CustomerOrganizationID, licenseKey.ID,
		); err != nil {
			return nil, fmt.Errorf("failed to check deployment references for license key: %w", err)
		} else if len(referencing) > 0 {
			return respond.JSON(api.AffectedDeploymentsConflictResponse{AffectedDeployments: referencing}).
				WithStatus(http.StatusConflict), nil
		}
	}

	if err := db.DeleteLicenseKeyWithID(ctx, licenseKey.ID); err != nil {
		return nil, fmt.Errorf("error deleting license key: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func licenseKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authCtx := auth.Authentication.Require(ctx)
		licenseKeyID, err := uuid.Parse(r.PathValue("licenseKeyId"))
		if err != nil {
			http.Error(w, "licenseKeyId is not a valid UUID", http.StatusBadRequest)
			return
		}
		licenseKey, err := db.GetLicenseKeyByID(ctx, licenseKeyID)
		if errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get license key", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if licenseKey.OrganizationID != *authCtx.CurrentOrgID() {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if authCtx.CurrentCustomerOrgID() != nil &&
			(licenseKey.CustomerOrganizationID == nil || *licenseKey.CustomerOrganizationID != *authCtx.CurrentCustomerOrgID()) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if partnerOrgID := authCtx.CurrentPartnerOrgID(); partnerOrgID != nil {
			if licenseKey.CustomerOrganizationID == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			co, coErr := db.GetCustomerOrganizationByID(ctx, *licenseKey.CustomerOrganizationID)
			if coErr != nil || !util.PtrEq(co.PartnerOrganizationID, partnerOrgID) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		ctx = internalctx.WithLicenseKey(ctx, licenseKey)
		next.ServeHTTP(w, r.WithContext(ctx))
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
