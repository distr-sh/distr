package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/supportbundle"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

// SupportBundleScriptRouter handles endpoints called by the collect script.
// All endpoints use query-param token auth tied to the specific bundle.
func SupportBundleScriptRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Support Bundles"))

	r.Route("/{bundleId}", func(r chiopenapi.Router) {
		r.Use(auth.SupportBundleAuthentication.Middleware)

		type BundleIDRequest struct {
			BundleID uuid.UUID `path:"bundleId"`
		}

		r.Get("/collect-script", respond.Handler(getCollectScriptHandler)).
			With(option.Description("Get support bundle collect script")).
			With(option.Request(BundleIDRequest{})).
			With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))

		r.Post("/resources", respond.Handler(uploadSupportBundleResourceHandler)).
			With(option.Description("Upload a support bundle resource")).
			With(option.Request(struct {
				BundleIDRequest
				Name        string                                 `formData:"name"`
				ContentType types.SupportBundleResourceContentType `formData:"contentType"`
				Content     string                                 `formData:"content"`
			}{}, option.ContentType("multipart/form-data"))).
			With(option.Response(http.StatusOK, api.SupportBundleResourceSummary{}))

		r.Post("/finalize", respond.Handler(finalizeSupportBundleHandler)).
			With(option.Description("Finalize a support bundle")).
			With(option.Request(BundleIDRequest{}))
	})
}

func getCollectScriptHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	bundle := auth.SupportBundleAuthentication.Require(ctx)

	bundleSecret := r.URL.Query().Get("bundleSecret")

	if org, err := db.GetOrganizationWithBranding(ctx, bundle.OrganizationID); err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	} else {
		baseURL := customdomains.AppDomainOrDefault(ctx, org.ID, org.Branding)

		if envVars, err := db.GetSupportBundleConfigurationEnvVars(ctx, bundle.OrganizationID); err != nil {
			return nil, fmt.Errorf("failed to get support bundle config env vars: %w", err)
		} else if scripts, err := db.GetEnabledSupportBundleConfigurationScripts(
			ctx, bundle.OrganizationID, bundle.DeploymentType,
		); err != nil {
			return nil, fmt.Errorf("failed to get support bundle config scripts: %w", err)
		} else if script, err := supportbundle.GenerateCollectScript(supportbundle.CollectScriptParams{
			BaseURL:          baseURL,
			BundleID:         bundle.ID,
			BundleSecret:     bundleSecret,
			EnvVars:          envVars,
			Scripts:          scripts,
			LogTailLines:     env.SupportBundleLogTailLines(),
			ResourceMaxBytes: env.SupportBundleResourceMaxBytes(),
			ScriptTimeout:    env.SupportBundleScriptTimeout(),
		}); err != nil {
			return nil, fmt.Errorf("failed to generate collect script: %w", err)
		} else {
			return respond.Stream("text/plain; charset=utf-8", strings.NewReader(script)), nil
		}
	}
}

func uploadSupportBundleResourceHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	bundle := auth.SupportBundleAuthentication.Require(ctx)

	// The request body size is limited by the requestSize middleware on the route. A multipart form
	// value does not go through decodeJSON, so it is not trimmed generically.
	if bundle.Status != types.SupportBundleStatusInitialized {
		return respond.BadRequest("support bundle is not accepting data"), nil
	} else if err := r.ParseMultipartForm(1 << 20); err != nil {
		return respond.Error(http.StatusRequestEntityTooLarge, "request body too large"), nil
	} else if name := strings.TrimSpace(r.FormValue("name")); name == "" {
		return respond.BadRequest("name is required"), nil
	} else if contentType, err := uploadedSupportBundleResourceContentType(r); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if file, _, err := r.FormFile("content"); err != nil {
		return respond.BadRequest("content file is required"), nil
	} else {
		defer file.Close()
		if contentBytes, err := io.ReadAll(file); err != nil {
			return respond.BadRequest("failed to read content"), nil
		} else if secrets, err := db.GetSecretsForCustomer(ctx, bundle.CustomerOrganizationID); err != nil {
			return nil, fmt.Errorf("failed to get secrets: %w", err)
		} else {
			resource := types.SupportBundleResource{
				SupportBundleID: bundle.ID,
				Name:            name,
				ContentType:     contentType,
				Content:         dbcrypto.String(secretsRedactor(secrets).Redact(string(contentBytes))),
			}
			if err := db.CreateSupportBundleResource(ctx, &resource); err != nil {
				return nil, fmt.Errorf("failed to create support bundle resource: %w", err)
			} else {
				return respond.JSON(mapping.SupportBundleResourceToSummaryAPI(resource)), nil
			}
		}
	}
}

// uploadedSupportBundleResourceContentType treats an upload without a content type as plain text,
// which the Docker collect script relies on.
func uploadedSupportBundleResourceContentType(r *http.Request) (types.SupportBundleResourceContentType, error) {
	if value := r.FormValue("contentType"); value == "" {
		return types.SupportBundleResourceContentTypeText, nil
	} else {
		return types.ParseSupportBundleResourceContentType(value)
	}
}

func finalizeSupportBundleHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	bundle := auth.SupportBundleAuthentication.Require(ctx)

	if bundle.Status != types.SupportBundleStatusInitialized {
		return respond.BadRequest("support bundle is not in initialized state"), nil
	} else if err := db.RunTxRR(ctx, func(ctx context.Context) error {
		if err := db.UpdateSupportBundleStatus(
			ctx, bundle.ID, bundle.OrganizationID, types.SupportBundleStatusCreated, nil,
		); err != nil {
			return err
		}

		return db.ClearSupportBundleBundleSecret(ctx, bundle.ID)
	}); err != nil {
		return nil, fmt.Errorf("failed to finalize support bundle: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}
