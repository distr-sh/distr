package handlers

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func SupportBundlesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Support Bundles"))
	r.Use(middleware.SupportBundlesCustomerFeatureMiddleware)

	r.With(middleware.RequireVendor, middleware.RequireOrgAndRole).Route("/configuration", func(r chiopenapi.Router) {
		r.Get("/", respond.Handler(getSupportBundleConfigurationHandler)).
			With(option.Description("Get support bundle configuration")).
			With(option.Response(http.StatusOK, []api.SupportBundleConfigurationEnvVar{}))

		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
			r.Put("/", respond.Handler(createOrUpdateSupportBundleConfigurationHandler)).
				With(option.Description("Create or update support bundle configuration")).
				With(option.Request(api.CreateUpdateSupportBundleConfigurationRequest{})).
				With(option.Response(http.StatusOK, []api.SupportBundleConfigurationEnvVar{}))
		})

		r.Route("/scripts", func(r chiopenapi.Router) {
			r.Get("/", respond.Handler(getSupportBundleConfigurationScriptsHandler)).
				With(option.Description("List support bundle custom scripts")).
				With(option.Response(http.StatusOK, []api.SupportBundleConfigurationScript{}))

			r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
				type ScriptIDRequest struct {
					ScriptID uuid.UUID `path:"scriptId"`
				}

				r.Post("/", respond.Handler(createSupportBundleConfigurationScriptHandler)).
					With(option.Description("Create a support bundle custom script")).
					With(option.Request(api.CreateUpdateSupportBundleConfigurationScriptRequest{})).
					With(option.Response(http.StatusOK, api.SupportBundleConfigurationScript{}))

				r.Put("/{scriptId}", respond.Handler(updateSupportBundleConfigurationScriptHandler)).
					With(option.Description("Update a support bundle custom script")).
					With(option.Request(struct {
						ScriptIDRequest
						api.CreateUpdateSupportBundleConfigurationScriptRequest
					}{})).
					With(option.Response(http.StatusOK, api.SupportBundleConfigurationScript{}))

				r.Delete("/{scriptId}", respond.Handler(deleteSupportBundleConfigurationScriptHandler)).
					With(option.Description("Delete a support bundle custom script")).
					With(option.Request(ScriptIDRequest{}))
			})
		})
	})

	r.With(middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Get("/", respond.Handler(getSupportBundlesHandler)).
			With(option.Description("List support bundles")).
			With(option.Response(http.StatusOK, []api.SupportBundle{}))

		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Post("/", respond.Handler(createSupportBundleHandler)).
			With(option.Description("Create a new support bundle")).
			With(option.Request(api.CreateSupportBundleRequest{})).
			With(option.Response(http.StatusOK, api.CreateSupportBundleResponse{}))

		r.Route("/{bundleId}", func(r chiopenapi.Router) {
			type BundleIDRequest struct {
				BundleID uuid.UUID `path:"bundleId"`
			}

			r.Get("/", respond.Handler(getSupportBundleDetailHandler)).
				With(option.Description("Get support bundle detail")).
				With(option.Request(BundleIDRequest{})).
				With(option.Response(http.StatusOK, api.SupportBundleDetail{}))

			r.Get("/download", respond.Handler(downloadSupportBundleResourcesHandler)).
				With(option.Description("Download all support bundle resources as a zip archive")).
				With(option.Request(BundleIDRequest{})).
				With(option.Response(http.StatusOK, nil, option.ContentType("application/zip")))

			r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Patch("/status", respond.Handler(updateSupportBundleStatusHandler)).
				With(option.Description("Update support bundle status")).
				With(option.Request(struct {
					BundleIDRequest
					api.UpdateSupportBundleStatusRequest
				}{}))

			r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Delete("/", respond.Handler(deleteSupportBundleHandler)).
				With(option.Description("Delete a support bundle")).
				With(option.Request(BundleIDRequest{}))

			r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Post("/comments", respond.Handler(createSupportBundleCommentHandler)).
				With(option.Description("Create a support bundle comment")).
				With(option.Request(struct {
					BundleIDRequest
					api.CreateSupportBundleCommentRequest
				}{})).
				With(option.Response(http.StatusOK, api.SupportBundleComment{}))
		})
	})
}

// Configuration handlers

func getSupportBundleConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if envVars, err := db.GetSupportBundleConfigurationEnvVars(ctx, *a.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get support bundle config env vars: %w", err)
	} else {
		return respond.JSON(mapping.SupportBundleConfigurationEnvVarsToAPI(envVars)), nil
	}
}

func createOrUpdateSupportBundleConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if request, resp := decodeJSON[api.CreateUpdateSupportBundleConfigurationRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		envVars := make([]types.SupportBundleConfigurationEnvVar, len(request.EnvVars))
		for i, ev := range request.EnvVars {
			envVars[i] = types.SupportBundleConfigurationEnvVar{
				OrganizationID: *a.CurrentOrgID(),
				Name:           ev.Name,
				Redacted:       ev.Redacted,
			}
		}

		if err := db.SaveSupportBundleConfigurationEnvVars(ctx, *a.CurrentOrgID(), envVars); err != nil {
			return nil, fmt.Errorf("failed to save support bundle configuration: %w", err)
		} else if savedEnvVars, err := db.GetSupportBundleConfigurationEnvVars(ctx, *a.CurrentOrgID()); err != nil {
			return nil, fmt.Errorf("failed to get support bundle config env vars: %w", err)
		} else {
			return respond.JSON(mapping.SupportBundleConfigurationEnvVarsToAPI(savedEnvVars)), nil
		}
	}
}

func getSupportBundleConfigurationScriptsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if scripts, err := db.GetSupportBundleConfigurationScripts(ctx, *a.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get support bundle config scripts: %w", err)
	} else {
		return respond.JSON(mapping.List(scripts, mapping.SupportBundleConfigurationScriptToAPI)), nil
	}
}

func createSupportBundleConfigurationScriptHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if request, resp := decodeJSON[api.CreateUpdateSupportBundleConfigurationScriptRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		script := mapping.SupportBundleConfigurationScriptToInternal(request, *a.CurrentOrgID())
		if err := db.CreateSupportBundleConfigurationScript(ctx, &script); err != nil {
			return respondSupportBundleConfigurationScriptError(err)
		} else {
			return respond.JSON(mapping.SupportBundleConfigurationScriptToAPI(script)), nil
		}
	}
}

func updateSupportBundleConfigurationScriptHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if scriptID, err := uuid.Parse(r.PathValue("scriptId")); err != nil {
		return supportBundleScriptNotFound, nil
	} else if request, resp := decodeJSON[api.CreateUpdateSupportBundleConfigurationScriptRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		script := mapping.SupportBundleConfigurationScriptToInternal(request, *a.CurrentOrgID())
		script.ID = scriptID
		if err := db.UpdateSupportBundleConfigurationScript(ctx, &script); err != nil {
			return respondSupportBundleConfigurationScriptError(err)
		} else {
			return respond.JSON(mapping.SupportBundleConfigurationScriptToAPI(script)), nil
		}
	}
}

func deleteSupportBundleConfigurationScriptHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if scriptID, err := uuid.Parse(r.PathValue("scriptId")); err != nil {
		return supportBundleScriptNotFound, nil
	} else if err := db.DeleteSupportBundleConfigurationScript(ctx, scriptID, *a.CurrentOrgID()); err != nil {
		return respondSupportBundleConfigurationScriptError(err)
	} else {
		return respond.NoContent(), nil
	}
}

func respondSupportBundleConfigurationScriptError(err error) (respond.Responder, error) {
	if errors.Is(err, distrerr.ErrNotFound) {
		return supportBundleScriptNotFound, nil
	} else if errors.Is(err, distrerr.ErrConflict) {
		return respond.Conflict("a script with this name already exists"), nil
	} else {
		return nil, fmt.Errorf("failed to save support bundle config script: %w", err)
	}
}

// Bundle handlers

func getSupportBundlesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if bundles, err := db.GetSupportBundles(
		ctx, *a.CurrentOrgID(), a.CurrentCustomerOrgID(), a.CurrentPartnerOrgID(),
	); err != nil {
		return nil, fmt.Errorf("failed to get support bundles: %w", err)
	} else {
		return respond.JSON(mapping.List(bundles, mapping.SupportBundleToAPI)), nil
	}
}

func createSupportBundleHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if a.CurrentCustomerOrgID() == nil {
		return respond.Forbidden("only customers can create support bundles"), nil
	} else if request, resp := decodeJSON[api.CreateSupportBundleRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if bundleSecret, err := security.GenerateAccessKey(); err != nil {
		return nil, fmt.Errorf("failed to generate collect token: %w", err)
	} else if org, err := db.GetOrganizationWithBranding(ctx, *a.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	} else {
		baseURL := customdomains.AppDomainOrDefault(ctx, org.ID, org.Branding)

		expiresAt := time.Now().UTC().Add(24 * time.Hour)
		bundle := types.SupportBundle{
			OrganizationID:         *a.CurrentOrgID(),
			CustomerOrganizationID: *a.CurrentCustomerOrgID(),
			CreatedByUserAccountID: new(a.CurrentUserID()),
			DeploymentType:         request.DeploymentType,
			Title:                  request.Title,
			Description:            request.Description,
			BundleSecret:           dbcrypto.String(bundleSecret),
			BundleSecretExpiresAt:  &expiresAt,
		}
		if err := db.CreateSupportBundle(ctx, &bundle); err != nil {
			return nil, fmt.Errorf("failed to create support bundle: %w", err)
		} else if detailBundle, err := db.GetSupportBundleByID(ctx, bundle.ID, *a.CurrentOrgID()); err != nil {
			return nil, fmt.Errorf("failed to get support bundle: %w", err)
		} else {
			collectCommand := fmt.Sprintf(
				"curl -fsSL '%s/api/v1/support-bundle-collect/%s/collect-script?bundleSecret=%s' | sh",
				baseURL, bundle.ID.String(), bundleSecret,
			)
			return respond.JSON(api.CreateSupportBundleResponse{
				SupportBundle:  mapping.SupportBundleToAPI(*detailBundle),
				CollectCommand: collectCommand,
			}), nil
		}
	}
}

func getSupportBundleDetailHandler(r *http.Request) (respond.Responder, error) {
	id, err := uuid.Parse(r.PathValue("bundleId"))
	if err != nil {
		return supportBundleNotFound, nil
	}

	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	bundle, err := db.GetSupportBundleByID(ctx, id, *a.CurrentOrgID())
	if errors.Is(err, distrerr.ErrNotFound) {
		return supportBundleNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get support bundle: %w", err)
	}

	if a.CurrentCustomerOrgID() != nil && bundle.CustomerOrganizationID != *a.CurrentCustomerOrgID() {
		return supportBundleNotFound, nil
	}
	if partnerOrgID := a.CurrentPartnerOrgID(); partnerOrgID != nil {
		err := db.ValidateCustomerOrgBelongsToPartnerOrg(ctx, bundle.CustomerOrganizationID, *partnerOrgID)
		if errors.Is(err, db.ErrCustomerOrgNotInPartnerOrg) {
			return supportBundleNotFound, nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to validate customer org belongs to partner org: %w", err)
		}
	}

	if resources, err := db.GetSupportBundleResources(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to get support bundle resources: %w", err)
	} else if comments, err := db.GetSupportBundleComments(ctx, id); err != nil {
		return nil, fmt.Errorf("failed to get support bundle comments: %w", err)
	} else {
		detail := api.SupportBundleDetail{
			SupportBundle: mapping.SupportBundleToAPI(*bundle),
			Resources:     mapping.List(resources, mapping.SupportBundleResourceToAPI),
			Comments:      mapping.List(comments, mapping.SupportBundleCommentToAPI),
		}
		if bundle.Status == types.SupportBundleStatusInitialized && bundle.BundleSecretExpiresAt != nil {
			org, err := db.GetOrganizationWithBranding(ctx, bundle.OrganizationID)
			if err != nil {
				return nil, fmt.Errorf("failed to get organization: %w", err)
			}
			baseURL := customdomains.AppDomainOrDefault(ctx, org.ID, org.Branding)
			cmd := fmt.Sprintf(
				"curl -fsSL '%s/api/v1/support-bundle-collect/%s/collect-script?bundleSecret=%s' | sh",
				baseURL, bundle.ID.String(), bundle.BundleSecret,
			)
			detail.CollectCommand = &cmd
		}
		return respond.JSON(detail), nil
	}
}

func downloadSupportBundleResourcesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()

	if bundle, resp, err := requireSupportBundle(r); resp != nil || err != nil {
		return resp, err
	} else if resources, err := db.GetSupportBundleResources(ctx, bundle.ID); err != nil {
		return nil, fmt.Errorf("failed to get support bundle resources: %w", err)
	} else {
		return respond.Writer("application/zip", func(w io.Writer) (respond.Responder, error) {
			if err := writeSupportBundleZip(w, resources); err != nil {
				return nil, fmt.Errorf("failed to write zip archive: %w", err)
			}
			return nil, nil
		}).WithHeader("Content-Disposition", attachment(supportBundleZipFileName(bundle))), nil
	}
}

func writeSupportBundleZip(w io.Writer, resources []types.SupportBundleResource) (err error) {
	zipWriter := zip.NewWriter(w)
	defer func() {
		if closeErr := zipWriter.Close(); err == nil {
			err = closeErr
		}
	}()

	usedNames := make(map[string]struct{})
	for _, resource := range resources {
		extension := resource.ContentType.FileExtension()
		name := strings.TrimSuffix(supportBundleZipEntryPath(resource.Name, resource.ID.String()), extension)
		entryName := name + extension
		for count := 2; ; count++ {
			if _, exists := usedNames[entryName]; !exists {
				break
			}
			entryName = fmt.Sprintf("%s-%d%s", name, count, extension)
		}
		usedNames[entryName] = struct{}{}
		entry, err := zipWriter.CreateHeader(&zip.FileHeader{
			Name:     entryName,
			Method:   zip.Deflate,
			Modified: resource.CreatedAt,
		})
		if err != nil {
			return err
		}
		if _, err := entry.Write([]byte(resource.Content)); err != nil {
			return err
		}
	}
	return nil
}

const supportBundleZipSegmentMaxBytes = 128

// supportBundleZipEntryPath turns a resource name into a relative path that cannot escape the
// directory the archive is extracted into. Extractors on Windows also split at a backslash and read
// a colon as a drive letter, so a backslash separates segments and a colon is replaced.
func supportBundleZipEntryPath(name, fallback string) string {
	var segments []string
	for segment := range strings.FieldsFuncSeq(name, func(r rune) bool { return r == '/' || r == '\\' }) {
		segment = strings.TrimSpace(truncateUTF8(strings.ReplaceAll(segment, ":", "-"), supportBundleZipSegmentMaxBytes))
		if segment != "" && segment != "." && segment != ".." {
			segments = append(segments, segment)
		}
	}
	if len(segments) == 0 {
		return fallback
	}
	return strings.Join(segments, "/")
}

func truncateUTF8(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := 0
	for i := range s {
		if i > maxBytes {
			break
		}
		cut = i
	}
	return s[:cut]
}

func supportBundleZipFileName(bundle *types.SupportBundleWithDetails) string {
	parts := []string{"distr-support-bundle"}
	if customer := zipFileNamePart(bundle.CustomerOrganizationName); customer != "" {
		parts = append(parts, customer)
	}
	if title := zipFileNamePart(bundle.Title); title != "" {
		parts = append(parts, title)
	}
	parts = append(parts, bundle.ID.String()[:8])
	return strings.Join(parts, "-") + ".zip"
}

// zipFileNamePart reduces a string to lowercase letters only, capped at 16 characters.
func zipFileNamePart(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
			if b.Len() >= 16 {
				break
			}
		}
	}
	return b.String()
}

func updateSupportBundleStatusHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	request, resp := decodeJSON[api.UpdateSupportBundleStatusRequest](r)
	if resp != nil {
		return resp, nil
	}

	var bundle *types.SupportBundleWithDetails
	var err error
	status := types.SupportBundleStatus(request.Status)
	switch status {
	case types.SupportBundleStatusResolved:
		if a.CurrentCustomerOrgID() != nil {
			return respond.Forbidden("customers cannot resolve support bundles"), nil
		}
		bundle, resp, err = requireSupportBundle(r)
		if resp != nil || err != nil {
			return resp, err
		}
		if bundle.Status == types.SupportBundleStatusResolved || bundle.Status == types.SupportBundleStatusCanceled {
			return respond.BadRequest("cannot resolve support bundle in its current state"), nil
		}
		changedBy := a.CurrentUserID()
		if bundle.Status == types.SupportBundleStatusInitialized {
			err = db.RunTxRR(ctx, func(ctx context.Context) error {
				if err := db.UpdateSupportBundleStatus(
					ctx, bundle.ID, bundle.OrganizationID, status, &changedBy,
				); err != nil {
					return err
				}
				return db.ClearSupportBundleBundleSecret(ctx, bundle.ID)
			})
		} else {
			err = db.UpdateSupportBundleStatus(
				ctx, bundle.ID, bundle.OrganizationID, status, &changedBy,
			)
		}
	case types.SupportBundleStatusCanceled:
		bundle, resp, err = requireSupportBundle(r)
		if resp != nil || err != nil {
			return resp, err
		}
		if bundle.Status != types.SupportBundleStatusInitialized {
			return respond.BadRequest("only initialized bundles can be canceled"), nil
		}
		changedBy := a.CurrentUserID()
		err = db.RunTxRR(ctx, func(ctx context.Context) error {
			if err := db.UpdateSupportBundleStatus(
				ctx, bundle.ID, bundle.OrganizationID, status, &changedBy,
			); err != nil {
				return err
			}
			return db.ClearSupportBundleBundleSecret(ctx, bundle.ID)
		})
	default:
		return respond.BadRequest("only 'resolved' or 'canceled' status is allowed"), nil
	}

	if errors.Is(err, distrerr.ErrNotFound) {
		return supportBundleNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to update support bundle status: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func deleteSupportBundleHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()

	if bundle, resp, err := requireSupportBundle(r); resp != nil || err != nil {
		return resp, err
	} else if err := db.DeleteSupportBundle(
		ctx, bundle.ID, bundle.OrganizationID,
	); errors.Is(err, distrerr.ErrNotFound) {
		return supportBundleNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to delete support bundle: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

// requireSupportBundle parses the bundle ID from the path, verifies org ownership
// and customer org access.
func requireSupportBundle(r *http.Request) (*types.SupportBundleWithDetails, respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if id, err := uuid.Parse(r.PathValue("bundleId")); err != nil {
		return nil, supportBundleNotFound, nil
	} else if bundle, err := db.GetSupportBundleByID(ctx, id, *a.CurrentOrgID()); errors.Is(err, distrerr.ErrNotFound) {
		return nil, supportBundleNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get support bundle: %w", err)
	} else if a.CurrentCustomerOrgID() != nil && bundle.CustomerOrganizationID != *a.CurrentCustomerOrgID() {
		return nil, supportBundleNotFound, nil
	} else if partnerOrgID := a.CurrentPartnerOrgID(); partnerOrgID == nil {
		return bundle, nil, nil
	} else if err := db.ValidateCustomerOrgBelongsToPartnerOrg(
		ctx, bundle.CustomerOrganizationID, *partnerOrgID,
	); errors.Is(err, db.ErrCustomerOrgNotInPartnerOrg) {
		return nil, supportBundleNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to validate customer org belongs to partner org: %w", err)
	} else {
		return bundle, nil, nil
	}
}

func createSupportBundleCommentHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)

	if bundle, resp, err := requireSupportBundle(r); resp != nil || err != nil {
		return resp, err
	} else if request, resp := decodeJSON[api.CreateSupportBundleCommentRequest](r); resp != nil {
		return resp, nil
	} else if request.Content == "" {
		return respond.BadRequest("content is required"), nil
	} else if comment, err := db.CreateSupportBundleComment(
		ctx, bundle.ID, a.CurrentUserID(), request.Content,
	); err != nil {
		return nil, fmt.Errorf("failed to create support bundle comment: %w", err)
	} else {
		return respond.JSON(mapping.SupportBundleCommentToAPI(*comment)), nil
	}
}
