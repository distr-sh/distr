package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func SupportBundlesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Support Bundles"))

	r.With(middleware.RequireVendor, middleware.RequireOrgAndRole).Route("/configuration", func(r chiopenapi.Router) {
		r.Get("/", getSupportBundleConfigurationHandler()).
			With(option.Description("Get support bundle configuration"))

		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
			r.Put("/", createOrUpdateSupportBundleConfigurationHandler()).
				With(option.Description("Create or update support bundle configuration")).
				With(option.Request(api.CreateUpdateSupportBundleConfigurationRequest{}))

			r.Delete("/", deleteSupportBundleConfigurationHandler()).
				With(option.Description("Delete support bundle configuration"))
		})
	})

	r.With(middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", getSupportBundlesHandler()).
			With(option.Description("List support bundles")).
			With(option.Response(http.StatusOK, []api.SupportBundle{}))

		r.Post("/", createSupportBundleHandler()).
			With(option.Description("Create a new support bundle")).
			With(option.Request(api.CreateSupportBundleRequest{})).
			With(option.Response(http.StatusOK, api.CreateSupportBundleResponse{}))

		r.Route("/{supportBundleId}", func(r chiopenapi.Router) {
			type SupportBundleIDRequest struct {
				SupportBundleID uuid.UUID `path:"supportBundleId"`
			}

			r.Get("/", getSupportBundleDetailHandler()).
				With(option.Description("Get support bundle detail")).
				With(option.Request(SupportBundleIDRequest{})).
				With(option.Response(http.StatusOK, api.SupportBundleDetail{}))

			r.With(middleware.RequireVendor, middleware.BlockSuperAdmin).
				Patch("/status", updateSupportBundleStatusHandler()).
				With(option.Description("Update support bundle status")).
				With(option.Request(struct {
					SupportBundleIDRequest
					api.UpdateSupportBundleStatusRequest
				}{}))

			r.With(middleware.BlockSuperAdmin).
				Post("/comments", createSupportBundleCommentHandler()).
				With(option.Description("Create a support bundle comment")).
				With(option.Request(struct {
					SupportBundleIDRequest
					api.CreateSupportBundleCommentRequest
				}{})).
				With(option.Response(http.StatusOK, api.SupportBundleComment{}))
		})
	})
}

// Configuration handlers

func getSupportBundleConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		envVars, err := db.GetSupportBundleConfigurationEnvVars(ctx, *a.CurrentOrgID())
		if err != nil {
			log.Error("failed to get support bundle config env vars", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.SupportBundleConfigurationEnvVarsToAPI(envVars))
	}
}

func createOrUpdateSupportBundleConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.CreateUpdateSupportBundleConfigurationRequest](w, r)
		if err != nil {
			return
		} else if err := request.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		envVars := make([]types.SupportBundleConfigurationEnvVar, len(request.EnvVars))
		for i, ev := range request.EnvVars {
			envVars[i] = types.SupportBundleConfigurationEnvVar{
				OrganizationID: *a.CurrentOrgID(),
				Name:           ev.Name,
				Redacted:       ev.Redacted,
			}
		}

		if err := db.SaveSupportBundleConfigurationEnvVars(ctx, *a.CurrentOrgID(), envVars); err != nil {
			log.Error("failed to save support bundle configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		savedEnvVars, err := db.GetSupportBundleConfigurationEnvVars(ctx, *a.CurrentOrgID())
		if err != nil {
			log.Error("failed to get support bundle config env vars", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.SupportBundleConfigurationEnvVarsToAPI(savedEnvVars))
	}
}

func deleteSupportBundleConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		if err := db.DeleteSupportBundleConfigurationEnvVars(ctx, *a.CurrentOrgID()); err != nil {
			log.Error("failed to delete support bundle configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// Bundle handlers

func getSupportBundlesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		bundles, err := db.GetSupportBundles(ctx, *a.CurrentOrgID(), a.CurrentCustomerOrgID())
		if err != nil {
			log.Error("failed to get support bundles", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.List(bundles, mapping.SupportBundleToAPI))
	}
}

func generateCollectToken() (raw string, hash []byte, err error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("could not generate collect token: %w", err)
	}
	raw = hex.EncodeToString(tokenBytes)
	h := sha256.Sum256(tokenBytes)
	return raw, h[:], nil
}

func createSupportBundleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		if a.CurrentCustomerOrgID() == nil {
			http.Error(w, "only customers can create support bundles", http.StatusForbidden)
			return
		}

		request, err := JsonBody[api.CreateSupportBundleRequest](w, r)
		if err != nil {
			return
		}

		if request.Title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		exists, err := db.ExistsSupportBundleConfigurationEnvVars(ctx, *a.CurrentOrgID())
		if err != nil {
			log.Error("failed to check support bundle configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "support bundle configuration not set up by vendor", http.StatusBadRequest)
			return
		}

		rawToken, tokenHash, err := generateCollectToken()
		if err != nil {
			log.Error("failed to generate collect token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		expiresAt := time.Now().Add(24 * time.Hour)
		bundle := types.SupportBundle{
			OrganizationID:         *a.CurrentOrgID(),
			CustomerOrganizationID: *a.CurrentCustomerOrgID(),
			CreatedByUserAccountID: a.CurrentUserID(),
			Title:                  request.Title,
			Description:            request.Description,
			CollectTokenHash:       tokenHash,
			CollectTokenExpiresAt:  &expiresAt,
		}
		if err := db.CreateSupportBundle(ctx, &bundle); err != nil {
			log.Error("failed to create support bundle", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		org, err := db.GetOrganizationByID(ctx, *a.CurrentOrgID())
		if err != nil {
			log.Error("failed to get organization", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		baseURL := customdomains.AppDomainOrDefault(*org)
		collectCommand := fmt.Sprintf(
			"curl -fsSL '%s/api/v1/support-bundle-collect/%s/collect-script?token=%s' | sh",
			baseURL, bundle.ID.String(), rawToken,
		)

		detailBundle, err := db.GetSupportBundleByID(ctx, bundle.ID, *a.CurrentOrgID())
		if err != nil {
			log.Error("failed to get support bundle", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, api.CreateSupportBundleResponse{
			SupportBundle:  mapping.SupportBundleToAPI(*detailBundle),
			CollectCommand: collectCommand,
		})
	}
}

func getSupportBundleDetailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		bundle, err := db.GetSupportBundleByID(ctx, id, *a.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Error("failed to get support bundle", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if a.CurrentCustomerOrgID() != nil && bundle.CustomerOrganizationID != *a.CurrentCustomerOrgID() {
			http.NotFound(w, r)
			return
		}

		resources, err := db.GetSupportBundleResources(ctx, id)
		if err != nil {
			log.Error("failed to get support bundle resources", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		comments, err := db.GetSupportBundleComments(ctx, id)
		if err != nil {
			log.Error("failed to get support bundle comments", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, api.SupportBundleDetail{
			SupportBundle: mapping.SupportBundleToAPI(*bundle),
			Resources:     mapping.List(resources, mapping.SupportBundleResourceToAPI),
			Comments:      mapping.List(comments, mapping.SupportBundleCommentToAPI),
		})
	}
}

func updateSupportBundleStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.UpdateSupportBundleStatusRequest](w, r)
		if err != nil {
			return
		}

		status := types.SupportBundleStatus(request.Status)
		if status != types.SupportBundleStatusResolved {
			http.Error(w, "only 'resolved' status is allowed", http.StatusBadRequest)
			return
		}

		resolvedBy := a.CurrentUserID()
		err = db.UpdateSupportBundleStatus(
			ctx, id, *a.CurrentOrgID(), status, &resolvedBy,
		)
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if err != nil {
			log.Error("failed to update support bundle status", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// requireSupportBundle parses the bundle ID from the path, verifies org ownership
// and customer org access. Returns nil if an error response was already written.
func requireSupportBundle(w http.ResponseWriter, r *http.Request) *types.SupportBundleWithDetails {
	id, err := uuid.Parse(r.PathValue("supportBundleId"))
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)

	bundle, err := db.GetSupportBundleByID(ctx, id, *a.CurrentOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
		return nil
	} else if err != nil {
		log.Error("failed to get support bundle", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	if a.CurrentCustomerOrgID() != nil && bundle.CustomerOrganizationID != *a.CurrentCustomerOrgID() {
		http.NotFound(w, r)
		return nil
	}

	return bundle
}

func createSupportBundleCommentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundle := requireSupportBundle(w, r)
		if bundle == nil {
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		a := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.CreateSupportBundleCommentRequest](w, r)
		if err != nil {
			return
		}

		if request.Content == "" {
			http.Error(w, "content is required", http.StatusBadRequest)
			return
		}

		comment, err := db.CreateSupportBundleComment(ctx, bundle.ID, a.CurrentUserID(), request.Content)
		if err != nil {
			log.Error("failed to create support bundle comment", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.SupportBundleCommentToAPI(*comment))
	}
}
