package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/supportbundle"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

// SupportBundleScriptRouter handles endpoints called by the collect script.
// All endpoints use query-param token auth tied to the specific bundle.
func SupportBundleScriptRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Support Bundles"))

	r.Route("/{bundleId}", func(r chiopenapi.Router) {
		r.Use(auth.SupportBundleAuthentication.Middleware)

		r.Get("/collect-script", getCollectScriptHandler()).
			With(option.Description("Get support bundle collect script")).
			With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))

		r.Post("/resources", uploadSupportBundleResourceHandler()).
			With(option.Description("Upload a support bundle resource")).
			With(option.Response(http.StatusOK, api.SupportBundleResourceSummary{}))

		r.Post("/finalize", finalizeSupportBundleHandler()).
			With(option.Description("Finalize a support bundle"))
	})
}

func getCollectScriptHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		bundle := auth.SupportBundleAuthentication.Require(ctx)

		bundleSecret := r.URL.Query().Get("bundleSecret")

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

		script, err := supportbundle.GenerateCollectScript(baseURL, bundle.ID, bundleSecret, envVars)
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
		bundle := auth.SupportBundleAuthentication.Require(ctx)

		if bundle.Status != types.SupportBundleStatusInitialized {
			http.Error(w, "support bundle is not accepting data", http.StatusBadRequest)
			return
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil {
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

		contentBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "failed to read content", http.StatusBadRequest)
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
		bundle := auth.SupportBundleAuthentication.Require(ctx)

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

			return db.ClearSupportBundleBundleSecret(ctx, bundle.ID)
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
