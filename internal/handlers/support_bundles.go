package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authkey"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/supportbundle"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
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

		r.With(middleware.RequireReadWriteOrAdmin).Group(func(r chiopenapi.Router) {
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

			r.Patch("/status", updateSupportBundleStatusHandler()).
				With(option.Description("Update support bundle status")).
				With(option.Request(struct {
					SupportBundleIDRequest
					api.UpdateSupportBundleStatusRequest
				}{}))

			r.Get("/resources", getSupportBundleResourcesHandler()).
				With(option.Description("List support bundle resources")).
				With(option.Request(SupportBundleIDRequest{})).
				With(option.Response(http.StatusOK, []api.SupportBundleResource{}))

			r.Get("/comments", getSupportBundleCommentsHandler()).
				With(option.Description("List support bundle comments")).
				With(option.Request(SupportBundleIDRequest{})).
				With(option.Response(http.StatusOK, []api.SupportBundleComment{}))

			r.Post("/comments", createSupportBundleCommentHandler()).
				With(option.Description("Create a support bundle comment")).
				With(option.Request(struct {
					SupportBundleIDRequest
					api.CreateSupportBundleCommentRequest
				}{})).
				With(option.Response(http.StatusOK, api.SupportBundleComment{}))
		})
	})
}

// SupportBundleScriptRouter handles endpoints called by the collect script.
// The collect-script endpoint uses query-param auth; the other endpoints use PAT header auth.
func SupportBundleScriptRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Support Bundles"))

	r.Route("/{supportBundleId}", func(r chiopenapi.Router) {
		r.With(queryAuthSupportBundleMiddleware).Group(func(r chiopenapi.Router) {
			r.Get("/collect-script", getCollectScriptHandler()).
				With(option.Description("Get support bundle collect script")).
				With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
		})

		r.With(auth.Authentication.Middleware).Group(func(r chiopenapi.Router) {
			r.Get("/config", getSupportBundleScriptConfigHandler()).
				With(option.Description("Get support bundle script configuration"))

			r.Post("/resources", uploadSupportBundleResourceHandler()).
				With(option.Description("Upload a support bundle resource")).
				With(option.Request(api.CreateSupportBundleResourceRequest{}))

			r.Post("/finalize", finalizeSupportBundleHandler()).
				With(option.Description("Finalize a support bundle"))
		})
	})
}

// Configuration handlers

func getSupportBundleConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		config, err := db.GetSupportBundleConfiguration(ctx, *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Error("failed to get support bundle configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		envVars, err := db.GetSupportBundleConfigurationEnvVars(ctx, config.ID)
		if err != nil {
			log.Error("failed to get support bundle config env vars", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.SupportBundleConfigurationToAPI(*config, envVars))
	}
}

func createOrUpdateSupportBundleConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.CreateUpdateSupportBundleConfigurationRequest](w, r)
		if err != nil {
			return
		}

		envVars := make([]types.SupportBundleConfigurationEnvVar, len(request.EnvVars))
		for i, ev := range request.EnvVars {
			envVars[i] = types.SupportBundleConfigurationEnvVar{
				Name:     ev.Name,
				Redacted: ev.Redacted,
			}
		}

		config, err := db.CreateOrUpdateSupportBundleConfiguration(ctx, *auth.CurrentOrgID(), envVars)
		if err != nil {
			log.Error("failed to create/update support bundle configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		savedEnvVars, err := db.GetSupportBundleConfigurationEnvVars(ctx, config.ID)
		if err != nil {
			log.Error("failed to get support bundle config env vars", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.SupportBundleConfigurationToAPI(*config, savedEnvVars))
	}
}

func deleteSupportBundleConfigurationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		if err := db.DeleteSupportBundleConfiguration(ctx, *auth.CurrentOrgID()); errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else if err != nil {
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
		auth := auth.Authentication.Require(ctx)

		bundles, err := db.GetSupportBundles(ctx, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID())
		if err != nil {
			log.Error("failed to get support bundles", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.List(bundles, mapping.SupportBundleToAPI))
	}
}

func createSupportBundleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		if auth.CurrentCustomerOrgID() == nil {
			http.Error(w, "only customers can create support bundles", http.StatusForbidden)
			return
		}

		request, err := JsonBody[api.CreateSupportBundleRequest](w, r)
		if err != nil {
			return
		}

		exists, err := db.ExistsSupportBundleConfiguration(ctx, *auth.CurrentOrgID())
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

		key, err := authkey.NewKey()
		if err != nil {
			log.Error("failed to generate PAT key", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		expiresAt := time.Now().Add(1 * time.Hour)
		token := types.AccessToken{
			ExpiresAt:      &expiresAt,
			Label:          util.PtrTo("support-bundle"),
			Key:            key,
			UserAccountID:  auth.CurrentUserID(),
			OrganizationID: *auth.CurrentOrgID(),
		}
		if err := db.CreateAccessToken(ctx, &token); err != nil {
			log.Error("failed to create PAT for support bundle", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		bundle := types.SupportBundle{
			OrganizationID:         *auth.CurrentOrgID(),
			CustomerOrganizationID: *auth.CurrentCustomerOrgID(),
			CreatedByUserAccountID: auth.CurrentUserID(),
			Title:                  request.Title,
			Description:            request.Description,
			AccessTokenID:          &token.ID,
		}
		if err := db.CreateSupportBundle(ctx, &bundle); err != nil {
			log.Error("failed to create support bundle", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		org, err := db.GetOrganizationByID(ctx, *auth.CurrentOrgID())
		if err != nil {
			log.Error("failed to get organization", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		baseURL := customdomains.AppDomainOrDefault(*org)
		collectCommand := fmt.Sprintf(
			"curl -fsSL '%s/api/v1/support-bundle-collect/%s/collect-script?token=%s' | sh",
			baseURL, bundle.ID.String(), key.String(),
		)

		detailBundle, err := db.GetSupportBundleByID(ctx, bundle.ID, *auth.CurrentOrgID())
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
		auth := auth.Authentication.Require(ctx)

		bundle, err := db.GetSupportBundleByID(ctx, id, *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Error("failed to get support bundle", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if auth.CurrentCustomerOrgID() != nil && bundle.CustomerOrganizationID != *auth.CurrentCustomerOrgID() {
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
		auth := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.UpdateSupportBundleStatusRequest](w, r)
		if err != nil {
			return
		}

		status := types.SupportBundleStatus(request.Status)
		if status != types.SupportBundleStatusResolved {
			http.Error(w, "only 'resolved' status is allowed", http.StatusBadRequest)
			return
		}

		if err := db.UpdateSupportBundleStatus(ctx, id, *auth.CurrentOrgID(), status); errors.Is(err, apierrors.ErrNotFound) {
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

func getSupportBundleResourcesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		resources, err := db.GetSupportBundleResources(ctx, id)
		if err != nil {
			log.Error("failed to get support bundle resources", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.List(resources, mapping.SupportBundleResourceToAPI))
	}
}

func getSupportBundleCommentsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		comments, err := db.GetSupportBundleComments(ctx, id)
		if err != nil {
			log.Error("failed to get support bundle comments", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.List(comments, mapping.SupportBundleCommentToAPI))
	}
}

func createSupportBundleCommentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		request, err := JsonBody[api.CreateSupportBundleCommentRequest](w, r)
		if err != nil {
			return
		}

		if request.Content == "" {
			http.Error(w, "content is required", http.StatusBadRequest)
			return
		}

		comment := types.SupportBundleComment{
			SupportBundleID: id,
			UserAccountID:   auth.CurrentUserID(),
			Content:         request.Content,
		}
		if err := db.CreateSupportBundleComment(ctx, &comment); err != nil {
			log.Error("failed to create support bundle comment", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, api.SupportBundleComment{
			ID:            comment.ID,
			CreatedAt:     comment.CreatedAt,
			UserAccountID: comment.UserAccountID,
			Content:       comment.Content,
		})
	}
}

// Script endpoints (called by the collect script)

func queryAuthSupportBundleMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			http.Error(w, "token is required", http.StatusUnauthorized)
			return
		}

		key, err := authkey.Parse(tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		tokenWithUser, err := db.GetAccessTokenByKeyUpdatingLastUsed(ctx, key)
		if err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			} else {
				log.Error("failed to validate support bundle token", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		bundleID, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.Error(w, "invalid bundle ID", http.StatusBadRequest)
			return
		}

		if _, err := db.GetSupportBundleByIDAndAccessToken(ctx, bundleID, tokenWithUser.ID); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.Error(w, "token does not match bundle", http.StatusUnauthorized)
			} else {
				log.Error("failed to verify support bundle token", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getCollectScriptHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		bundleID, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.Error(w, "invalid bundle ID", http.StatusBadRequest)
			return
		}

		tokenStr := r.URL.Query().Get("token")

		key, err := authkey.Parse(tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		tokenWithUser, err := db.GetAccessTokenByKeyUpdatingLastUsed(ctx, key)
		if err != nil {
			log.Error("failed to get access token", zap.Error(err))
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		org, err := db.GetOrganizationByID(ctx, tokenWithUser.OrganizationID)
		if err != nil {
			log.Error("failed to get organization", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		baseURL := customdomains.AppDomainOrDefault(*org)
		script := supportbundle.GenerateCollectScript(baseURL, bundleID, tokenStr)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := w.Write([]byte(script)); err != nil {
			log.Warn("failed to write collect script", zap.Error(err))
		}
	}
}

func getSupportBundleScriptConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		config, err := db.GetSupportBundleConfiguration(ctx, *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			RespondJSON(w, api.SupportBundleScriptConfig{EnvVars: []api.SupportBundleConfigurationEnvVar{}})
			return
		} else if err != nil {
			log.Error("failed to get support bundle configuration", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		envVars, err := db.GetSupportBundleConfigurationEnvVars(ctx, config.ID)
		if err != nil {
			log.Error("failed to get support bundle config env vars", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		apiEnvVars := make([]api.SupportBundleConfigurationEnvVar, len(envVars))
		for i, ev := range envVars {
			apiEnvVars[i] = api.SupportBundleConfigurationEnvVar{
				Name:     ev.Name,
				Redacted: ev.Redacted,
			}
		}

		RespondJSON(w, api.SupportBundleScriptConfig{EnvVars: apiEnvVars})
	}
}

func uploadSupportBundleResourceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundleID, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.Error(w, "invalid bundle ID", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		request, err := JsonBody[api.CreateSupportBundleResourceRequest](w, r)
		if err != nil {
			return
		}

		if request.Name == "" || request.Content == "" {
			http.Error(w, "name and content are required", http.StatusBadRequest)
			return
		}

		resource := types.SupportBundleResource{
			SupportBundleID: bundleID,
			Name:            request.Name,
			Content:         request.Content,
		}
		if err := db.CreateSupportBundleResource(ctx, &resource); err != nil {
			log.Error("failed to create support bundle resource", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.SupportBundleResourceToAPI(resource))
	}
}

func finalizeSupportBundleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundleID, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.Error(w, "invalid bundle ID", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		bundle, err := db.GetSupportBundleByID(ctx, bundleID, *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		} else if err != nil {
			log.Error("failed to get support bundle", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = db.UpdateSupportBundleStatus(
			ctx, bundleID, bundle.OrganizationID, types.SupportBundleStatusCreated,
		)
		if err != nil {
			log.Error("failed to update support bundle status", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if bundle.AccessTokenID != nil {
			if err := db.DeleteAccessToken(ctx, *bundle.AccessTokenID, bundle.CreatedByUserAccountID); err != nil {
				log.Warn("failed to delete support bundle PAT", zap.Error(err))
			}
			if err := db.ClearSupportBundleAccessToken(ctx, bundleID); err != nil {
				log.Warn("failed to clear support bundle access token reference", zap.Error(err))
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
