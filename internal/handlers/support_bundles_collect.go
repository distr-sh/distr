package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/supportbundle"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

// SupportBundleScriptRouter handles endpoints called by the collect script.
// All endpoints use query-param token auth tied to the specific bundle.
func SupportBundleScriptRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Support Bundles"))

	r.Route("/{supportBundleId}", func(r chiopenapi.Router) {
		r.With(collectTokenAuthMiddleware).Group(func(r chiopenapi.Router) {
			r.Get("/collect-script", getCollectScriptHandler()).
				With(option.Description("Get support bundle collect script")).
				With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))

			r.Post("/resources", uploadSupportBundleResourceHandler()).
				With(option.Description("Upload a support bundle resource")).
				With(option.Response(http.StatusOK, api.SupportBundleResourceSummary{}))

			r.Post("/finalize", finalizeSupportBundleHandler()).
				With(option.Description("Finalize a support bundle"))
		})
	})
}

type collectTokenContextKey struct{}

func collectTokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			http.Error(w, "token is required", http.StatusUnauthorized)
			return
		}

		tokenBytes, err := hex.DecodeString(tokenStr)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		h := sha256.Sum256(tokenBytes)
		tokenHash := h[:]

		bundleID, err := uuid.Parse(r.PathValue("supportBundleId"))
		if err != nil {
			http.Error(w, "invalid bundle ID", http.StatusBadRequest)
			return
		}

		bundle, err := db.GetSupportBundleByCollectToken(ctx, bundleID, tokenHash)
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		} else if err != nil {
			log.Error("failed to validate collect token", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx = context.WithValue(ctx, collectTokenContextKey{}, bundle)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bundleFromCollectToken(ctx context.Context) *types.SupportBundle {
	if bundle, ok := ctx.Value(collectTokenContextKey{}).(*types.SupportBundle); ok {
		return bundle
	}
	return nil
}

func getCollectScriptHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		bundle := bundleFromCollectToken(ctx)

		tokenStr := r.URL.Query().Get("token")

		org, err := db.GetOrganizationByID(ctx, bundle.OrganizationID)
		if err != nil {
			log.Error("failed to get organization", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		baseURL := customdomains.AppDomainOrDefault(*org)

		envVars, err := db.GetSupportBundleConfigurationEnvVars(ctx, bundle.OrganizationID)
		if err != nil {
			log.Error("failed to get support bundle config env vars", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		script, err := supportbundle.GenerateCollectScript(baseURL, bundle.ID, tokenStr, envVars)
		if err != nil {
			log.Error("failed to generate collect script", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := w.Write([]byte(script)); err != nil {
			log.Warn("failed to write collect script", zap.Error(err))
		}
	}
}

func uploadSupportBundleResourceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		bundle := bundleFromCollectToken(ctx)

		const maxSize = 5 * 1024 * 1024 // 5MB
		if err := r.ParseMultipartForm(maxSize); err != nil {
			http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
			return
		}

		name := r.FormValue("name")
		if name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("content")
		if err != nil {
			http.Error(w, "content file is required", http.StatusBadRequest)
			return
		}
		defer file.Close()

		contentBytes, err := io.ReadAll(io.LimitReader(file, maxSize+1))
		if err != nil {
			http.Error(w, "failed to read content", http.StatusBadRequest)
			return
		}
		if len(contentBytes) > maxSize {
			http.Error(w, "content exceeds maximum size of 5MB", http.StatusBadRequest)
			return
		}

		resource := types.SupportBundleResource{
			SupportBundleID: bundle.ID,
			Name:            name,
			Content:         string(contentBytes),
		}
		if err := db.CreateSupportBundleResource(ctx, &resource); err != nil {
			log.Error("failed to create support bundle resource", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		RespondJSON(w, mapping.SupportBundleResourceToSummaryAPI(resource))
	}
}

func finalizeSupportBundleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		bundle := bundleFromCollectToken(ctx)

		if bundle.Status != types.SupportBundleStatusInitialized {
			http.Error(w, "support bundle is not in initialized state", http.StatusBadRequest)
			return
		}

		err := db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.UpdateSupportBundleStatus(
				ctx, bundle.ID, bundle.OrganizationID, types.SupportBundleStatusCreated, nil,
			); err != nil {
				return err
			}

			return db.ClearSupportBundleCollectToken(ctx, bundle.ID)
		})
		if err != nil {
			log.Error("failed to finalize support bundle", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
