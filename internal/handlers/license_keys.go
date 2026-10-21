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
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/licensekey"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
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

	r.With(respond.Middleware(licenseKeyMiddleware)).Route("/{licenseKeyId}", func(r chiopenapi.Router) {
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
				r.Put("/", respond.Handler(updateLicenseKey)).
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
			); errors.Is(err, distrerr.ErrNotFound) {
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

		if _, resp, err := customerScope(ctx, body.CustomerOrganizationID); resp != nil || err != nil {
			return resp, err
		} else if err := db.CreateLicenseKey(ctx, &licenseKey); errors.Is(err, distrerr.ErrConflict) {
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

func updateLicenseKey(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	existing := internalctx.GetLicenseKey(ctx)
	authCtx := auth.Authentication.Require(ctx)

	body, resp := decodeJSON[api.UpdateLicenseKeyRequest](r)
	if resp != nil {
		return resp, nil
	}

	if body.LicenseTemplateID != nil {
		if _, err := db.GetLicenseTemplateByID(
			ctx, *body.LicenseTemplateID, *authCtx.CurrentOrgID(),
		); errors.Is(err, distrerr.ErrNotFound) {
			return respond.BadRequest("license template not found"), nil
		} else if err != nil {
			return nil, fmt.Errorf("could not get license template: %w", err)
		}
	}

	if body.Payload != nil {
		if err := licensekey.ValidatePayload(*body.Payload); err != nil {
			return respond.BadRequest(err.Error()), nil
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
		return respond.BadRequest("expiresAt must be after notBefore"), nil
	}

	needsRevision, err := licenseKeyRevisionChanged(existing, body)
	if err != nil {
		return nil, fmt.Errorf("could not compare license key revision: %w", err)
	}

	affected := []api.AffectedDeployment{}
	if needsRevision && existing.CustomerOrganizationID != nil {
		if hypotheticalLicenseKey, err := updatedLicenseKeyForRequest(existing, body); err != nil {
			return respond.BadRequest(err.Error()), nil
		} else if affected, err = findAffectedDeploymentsByLicenseKey(
			ctx,
			*authCtx.CurrentOrgID(),
			*existing.CustomerOrganizationID,
			hypotheticalLicenseKey,
		); err != nil {
			return deploymentValuesError(err, "invalid deployment values")
		}
	}

	if confirm := r.URL.Query().Get("confirm") == "true"; !confirm && len(affected) > 0 {
		return respond.JSON(api.AffectedDeploymentsConflictResponse{AffectedDeployments: affected}).
			WithStatus(http.StatusConflict), nil
	}

	var result *types.LicenseKey
	if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		var err error
		if result, err = db.UpdateLicenseKeyMetadata(
			ctx, existing.ID, body.Description, body.LicenseTemplateID,
		); err != nil {
			return nil, fmt.Errorf("could not update license key: %w", err)
		}

		if needsRevision {
			if resp, err := applyLicenseKeyRevision(ctx, existing, body, result); resp != nil || err != nil {
				return resp, err
			}
		}

		if err := triggerAffectedDeployments(
			ctx, affected, new(authCtx.CurrentUserID()), types.DeploymentRevisionTriggerLicenseKeyChange,
		); err != nil {
			return nil, fmt.Errorf("could not trigger affected deployments: %w", err)
		}
		return nil, nil
	}); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.JSON(api.UpdateLicenseKeyResponse{
			LicenseKey:          *result,
			AffectedDeployments: affected,
		}), nil
	}
}

func applyLicenseKeyRevision(
	ctx context.Context,
	existing *types.LicenseKey,
	body api.UpdateLicenseKeyRequest,
	result *types.LicenseKey,
) (respond.Responder, error) {
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
		return respond.BadRequest("notBefore, expiresAt and payload are required to create a revision"), nil
	}
	revision := types.LicenseKeyRevision{
		LicenseKeyID: existing.ID,
		NotBefore:    *notBefore,
		ExpiresAt:    *expiresAt,
		Payload:      payload,
	}
	if err := db.CreateLicenseKeyRevision(ctx, &revision); err != nil {
		return nil, fmt.Errorf("could not create license key revision: %w", err)
	}
	result.NotBefore = &revision.NotBefore
	result.ExpiresAt = &revision.ExpiresAt
	result.Payload = revision.Payload
	result.LastRevisedAt = &revision.CreatedAt
	return nil, nil
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

func licenseKeyMiddleware(r *http.Request) (*http.Request, respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)
	if licenseKeyID, err := uuid.Parse(r.PathValue("licenseKeyId")); err != nil {
		return nil, respond.BadRequest("licenseKeyId is not a valid UUID"), nil
	} else if licenseKey, err := db.GetLicenseKeyByID(ctx, licenseKeyID); errors.Is(err, distrerr.ErrNotFound) {
		return nil, licenseKeyNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get license key: %w", err)
	} else if licenseKey.OrganizationID != *authCtx.CurrentOrgID() {
		return nil, licenseKeyNotFound, nil
	} else if customerOrgID := authCtx.CurrentCustomerOrgID(); customerOrgID != nil &&
		!util.PtrEq(licenseKey.CustomerOrganizationID, customerOrgID) {
		return nil, licenseKeyNotFound, nil
	} else if resp, err := checkReachableByPartner(
		ctx, licenseKey.CustomerOrganizationID, licenseKeyNotFound,
	); resp != nil || err != nil {
		return nil, resp, err
	} else {
		return r.WithContext(internalctx.WithLicenseKey(ctx, licenseKey)), nil, nil
	}
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
