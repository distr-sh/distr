package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/notification"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ApplicationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Applications"))
	r.Use(middleware.RequireOrgAndRole)

	r.Get("/", respond.Handler(getApplications)).
		With(option.Description("List all applications")).
		With(option.Response(http.StatusOK, []api.ApplicationResponse{}))

	r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", respond.Handler(createApplication)).
		With(option.Description("Create a new application")).
		With(option.Request(api.CreateApplicationRequest{})).
		With(option.Response(http.StatusOK, api.ApplicationResponse{}))

	r.Route("/{applicationId}", func(r chiopenapi.Router) {
		type ApplicationRequest struct {
			ApplicationID string `path:"applicationId"`
		}

		r.With(applicationMiddleware).Group(func(r chiopenapi.Router) {
			r.Get("/", respond.Handler(getApplication)).
				With(option.Description("Get an application by ID")).
				With(option.Request(ApplicationRequest{})).
				With(option.Response(http.StatusOK, api.ApplicationResponse{}))
			r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Group(func(r chiopenapi.Router) {
					r.Delete("/", respond.Handler(deleteApplication)).
						With(option.Description("Delete an application")).
						With(option.Request(ApplicationRequest{}))
					r.Put("/", updateApplication).
						With(option.Description("Update an application")).
						With(option.Request(struct {
							ApplicationRequest
							api.UpdateApplicationRequest
						}{})).
						With(option.Response(http.StatusOK, api.ApplicationResponse{}))
					r.Patch("/", patchApplicationHandler()).
						With(option.Description("Partially update an application")).
						With(option.Request(struct {
							ApplicationRequest
							api.PatchApplicationRequest
						}{})).
						With(option.Response(http.StatusOK, api.ApplicationResponse{}))
					r.Patch("/image", patchImageApplication).
						With(option.Description("Update application image")).
						With(option.Request(struct {
							ApplicationRequest
							api.PatchImageRequest
						}{})).
						With(option.Response(http.StatusOK, api.ApplicationResponse{}))
				})
		})

		r.Route("/versions", func(r chiopenapi.Router) {
			// note that it would not be necessary to use the applicationMiddleware for the versions endpoints
			// it loads the application from the db including all versions, but I guess for now this is easier
			// when performance becomes more important, we should avoid this and do the request on the database layer
			r.With(applicationMiddleware).
				Group(func(r chiopenapi.Router) {
					r.With(middleware.RequireVendor).
						With(middleware.RequireReadWriteOrAdmin).
						With(middleware.BlockSuperAdmin).
						Post("/", createApplicationVersion).
						With(option.Description("Create a new application version")).
						With(option.Request(struct {
							ApplicationRequest
							api.CreateApplicationVersionRequest
						}{})).
						With(option.Response(http.StatusOK, api.ApplicationVersionResponse{}))
				})
			r.Route("/{applicationVersionId}", func(r chiopenapi.Router) {
				type ApplicationVersionRequest struct {
					ApplicationRequest
					ApplicationVersionId string `path:"applicationVersionId"`
				}

				r.With(applicationMiddleware).
					Get("/", respond.Handler(getApplicationVersion)).
					With(option.Description("Get an application version")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, api.ApplicationVersionResponse{}))
				r.With(middleware.RequireVendor).
					With(middleware.RequireReadWriteOrAdmin).
					With(middleware.BlockSuperAdmin).
					With(applicationMiddleware).
					Put("/", respond.Handler(updateApplicationVersion)).
					With(option.Description("Update an application version")).
					With(option.Request(struct {
						ApplicationVersionRequest
						api.UpdateApplicationVersionRequest
					}{})).
					With(option.Response(http.StatusOK, api.ApplicationVersionResponse{}))
				r.Get("/compose-file", getApplicationVersionComposeFile).
					With(option.Description("Get application version compose file")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, map[string]any{}, option.ContentType("application/yaml")))
				r.Get("/template-file", getApplicationVersionTemplateFile).
					With(option.Description("Get application version template file")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
				r.Get("/values-file", getApplicationVersionValuesFile).
					With(option.Description("Get application version values file")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, map[string]any{}, option.ContentType("application/yaml")))
				r.Get("/resources", respond.Handler(getApplicationVersionResources)).
					With(option.Description("Get application version resources")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, []types.ApplicationVersionResource{}))
			})
		})
	})
}

func applicationMapper(ctx context.Context) func(types.Application) api.ApplicationResponse {
	a := auth.Authentication.Require(ctx)
	return mapping.ApplicationToAPI(a.CurrentCustomerOrgID(), a.CurrentPartnerOrgID())
}

func applicationVersionMapper(ctx context.Context) func(types.ApplicationVersion) api.ApplicationVersionResponse {
	a := auth.Authentication.Require(ctx)
	return mapping.ApplicationVersionToAPI(a.CurrentCustomerOrgID(), a.CurrentPartnerOrgID())
}

// checkApplicationSettings checks the versioning strategy and the automatic update opt-in of an
// application about to be written. The error it returns is meant for the end user.
func checkApplicationSettings(
	org *types.Organization,
	application *types.Application,
	previousStrategy types.VersioningStrategy,
) error {
	if application.VersioningStrategy != previousStrategy {
		if !application.VersioningStrategy.IsSelectable() {
			return fmt.Errorf("versioning strategy must be one of %v", types.SelectableVersioningStrategies())
		}
		if err := types.ValidateVersionsForStrategy(application.VersioningStrategy, application.Versions); err != nil {
			return err
		}
	}
	if application.AllowAutomaticUpdates {
		if !org.HasFeature(types.FeatureAutoUpdates) {
			return errors.New("automatic updates are not enabled for this organization")
		}
		if !application.VersioningStrategy.AllowsAutomaticUpdates() {
			return errors.New("automatic updates require a versioning strategy of semver or chronological")
		}
	}
	return nil
}

// validateApplicationSettings writes the reason checkApplicationSettings rejects to the response.
func validateApplicationSettings(
	w http.ResponseWriter,
	org *types.Organization,
	application *types.Application,
	previousStrategy types.VersioningStrategy,
) error {
	if err := checkApplicationSettings(org, application, previousStrategy); err != nil {
		return badRequestError(w, err.Error())
	}
	return nil
}

func createApplication(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if request, resp := decodeJSON[api.CreateApplicationRequest](r); resp != nil {
		return resp, nil
	} else if request.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else {
		application := mapping.CreateApplicationToInternal(request)
		if application.VersioningStrategy == "" {
			application.VersioningStrategy = types.VersioningStrategyChronological
		}
		if err := checkApplicationSettings(auth.CurrentOrg(), &application, ""); err != nil {
			return respond.BadRequest(err.Error()), nil
		} else if err := db.CreateApplication(ctx, &application, *auth.CurrentOrgID()); err != nil {
			return nil, fmt.Errorf("could not create application: %w", err)
		} else {
			return respond.JSON(applicationMapper(ctx)(application)), nil
		}
	}
}

func updateApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	request, err := JsonBody[api.UpdateApplicationRequest](w, r)
	if err != nil {
		return
	} else if request.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	existing := internalctx.GetApplication(ctx)
	application := mapping.UpdateApplicationToInternal(request, *existing)
	// A client written before the strategy existed does not send it, and must not silently move
	// the application off the one it has.
	if application.VersioningStrategy == "" {
		application.VersioningStrategy = existing.VersioningStrategy
	}
	if validateApplicationSettings(w, auth.CurrentOrg(), &application, existing.VersioningStrategy) != nil {
		return
	}

	if !runTxOrRespond(ctx, w, func(ctx context.Context) error {
		if err := db.UpdateApplication(ctx, &application, *auth.CurrentOrgID()); err != nil {
			log.Warn("could not update application", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		// TODO ?
		// there surely is some way to have the update command returning the versions too, but I don't think it's worth
		// the work right now
		application.Versions = existing.Versions

		if err := triggerAutomaticApplicationUpdates(
			ctx, auth.CurrentOrg(), application.ID, new(auth.CurrentUserID()),
		); err != nil {
			return automaticUpdateError(ctx, w, err)
		}
		return nil
	}) {
		return
	}

	RespondJSON(w, applicationMapper(ctx)(application))
}

func patchApplicationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		existing := internalctx.GetApplication(ctx)
		patch, err := JsonBody[api.PatchApplicationRequest](w, r)
		if err != nil {
			return
		}

		previousStrategy := existing.VersioningStrategy
		patched := *existing
		if patch.Name != nil {
			patched.Name = *patch.Name
		}
		if patch.VersioningStrategy != nil {
			patched.VersioningStrategy = *patch.VersioningStrategy
		}
		if patch.AllowAutomaticUpdates != nil {
			patched.AllowAutomaticUpdates = *patch.AllowAutomaticUpdates
		}
		if validateApplicationSettings(w, auth.CurrentOrg(), &patched, previousStrategy) != nil {
			return
		}
		applicationNeedsUpdate := patched.Name != existing.Name ||
			patched.VersioningStrategy != existing.VersioningStrategy ||
			patched.AllowAutomaticUpdates != existing.AllowAutomaticUpdates

		if !runTxOrRespond(ctx, w, func(ctx context.Context) error {
			if applicationNeedsUpdate {
				versions := existing.Versions
				if err := db.UpdateApplication(ctx, &patched, *auth.CurrentOrgID()); err != nil {
					log.Warn("could not update application", zap.Error(err))
					sentry.GetHubFromContext(ctx).CaptureException(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return err
				}
				patched.Versions = versions
				*existing = patched
			}

			for _, vp := range patch.Versions {
				var ev *types.ApplicationVersion
				for i, v := range existing.Versions {
					if v.ID == vp.ID {
						ev = &existing.Versions[i]
						break
					}
				}
				if ev == nil {
					http.Error(w, fmt.Sprintf("no ApplicationVersion found with ID %v", vp.ID), http.StatusBadRequest)
					return errors.New("bad request")
				}

				versionNeedsUpdate := false
				if !util.PtrEq(ev.ArchivedAt, vp.ArchivedAt) {
					ev.ArchivedAt = vp.ArchivedAt
					versionNeedsUpdate = true
				}

				if versionNeedsUpdate {
					if err := db.UpdateApplicationVersion(ctx, ev); err != nil {
						log.Warn("could not update application version", zap.Error(err))
						sentry.GetHubFromContext(ctx).CaptureException(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return err
					}
				}
			}

			// Un-archiving a version, or switching the strategy, can make another version the
			// newest one.
			if err := triggerAutomaticApplicationUpdates(
				ctx, auth.CurrentOrg(), existing.ID, new(auth.CurrentUserID()),
			); err != nil {
				return automaticUpdateError(ctx, w, err)
			}
			return nil
		}) {
			return
		}

		RespondJSON(w, applicationMapper(ctx)(*existing))
	}
}

func getApplications(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	org := auth.CurrentOrg()
	var err error
	var applications []types.Application
	if org.HasFeature(types.FeatureLicensing) && auth.CurrentCustomerOrgID() != nil {
		// Get applications based on entitlement owner ID only if there is at least one entitlement in the parent organization
		entitlements, err1 := db.GetApplicationEntitlementsWithOrganizationID(ctx, *auth.CurrentOrgID(), nil)
		if err1 != nil {
			return nil, fmt.Errorf("failed to get application entitlements: %w", err1)
		} else if len(entitlements) > 0 {
			applications, err = db.GetApplicationsWithEntitlementOwnerID(ctx, *auth.CurrentCustomerOrgID())
		} else {
			applications, err = db.GetApplicationsByOrgID(ctx, *auth.CurrentOrgID())
		}
	} else {
		applications, err = db.GetApplicationsByOrgID(ctx, *auth.CurrentOrgID())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	} else {
		return respond.JSON(mapping.List(applications, applicationMapper(ctx))), nil
	}
}

var (
	applicationNotFound        = respond.NotFound("application not found")
	applicationVersionNotFound = respond.NotFound("application version not found")
)

func getApplication(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	org := auth.CurrentOrg()
	if org.HasFeature(types.FeatureLicensing) && auth.CurrentCustomerOrgID() != nil {
		if applicationID, err := uuid.Parse(r.PathValue("applicationId")); err != nil {
			return applicationNotFound, nil
		} else if entitlements, err := db.GetApplicationEntitlementsWithOrganizationID(
			ctx, *auth.CurrentOrgID(), nil,
		); err != nil {
			return nil, fmt.Errorf("failed to get application entitlements: %w", err)
		} else if len(entitlements) > 0 {
			application, err := db.GetApplicationWithEntitlementOwnerID(ctx, *auth.CurrentCustomerOrgID(), applicationID)
			if errors.Is(err, apierrors.ErrNotFound) {
				return applicationNotFound, nil
			} else if err != nil {
				return nil, fmt.Errorf("failed to get application: %w", err)
			} else {
				return respond.JSON(applicationMapper(ctx)(*application)), nil
			}
		} else {
			return respond.JSON(applicationMapper(ctx)(*internalctx.GetApplication(ctx))), nil
		}
	} else {
		return respond.JSON(applicationMapper(ctx)(*internalctx.GetApplication(ctx))), nil
	}
}

// getAccessibleApplicationVersion answers 404 for a version outside the current organization,
// outside the application in the path or, for a customer, not covered by one of their entitlements.
// Like getApplication, it applies entitlements only once the vendor has created any.
func getAccessibleApplicationVersion(r *http.Request) (*types.ApplicationVersion, respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if applicationID, err := uuid.Parse(r.PathValue("applicationId")); err != nil {
		return nil, applicationVersionNotFound, nil
	} else if applicationVersionID, err := uuid.Parse(r.PathValue("applicationVersionId")); err != nil {
		return nil, applicationVersionNotFound, nil
	} else {
		var entitledCustomerOrgID *uuid.UUID
		if auth.CurrentCustomerOrgID() != nil && auth.CurrentOrg().HasFeature(types.FeatureLicensing) {
			if hasEntitlements, err := db.HasAnyApplicationEntitlement(ctx, *auth.CurrentOrgID()); err != nil {
				return nil, nil, fmt.Errorf("failed to check for application entitlements: %w", err)
			} else if hasEntitlements {
				entitledCustomerOrgID = auth.CurrentCustomerOrgID()
			}
		}

		if version, err := db.GetApplicationVersionOfApplication(
			ctx, applicationVersionID, applicationID, *auth.CurrentOrgID(), entitledCustomerOrgID,
		); errors.Is(err, apierrors.ErrNotFound) {
			return nil, applicationVersionNotFound, nil
		} else if err != nil {
			return nil, nil, fmt.Errorf("failed to get application version: %w", err)
		} else {
			return version, nil, nil
		}
	}
}

func getApplicationVersion(r *http.Request) (respond.Responder, error) {
	if applicationVersion, resp, err := getAccessibleApplicationVersion(r); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.JSON(applicationVersionMapper(r.Context())(*applicationVersion)), nil
	}
}

func createApplicationVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	body := r.FormValue("applicationversion")
	var request api.CreateApplicationVersionRequest
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&request); err != nil {
		log.Error("failed to decode version", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	validation.TrimStrings(&request)

	application := internalctx.GetApplication(ctx)
	applicationVersion := mapping.CreateApplicationVersionToInternal(request)
	applicationVersion.ApplicationID = application.ID

	if application.Type == types.DeploymentTypeDocker {
		if data, ok := readMultipartFile(w, r, "composefile"); !ok {
			return
		} else {
			applicationVersion.ComposeFileData = data
			if _, err := applicationVersion.ParsedComposeFile(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if data, ok := readMultipartFile(w, r, "templatefile"); !ok {
			return
		} else {
			applicationVersion.TemplateFileData = data
		}
	} else {
		if data, ok := readMultipartFile(w, r, "valuesfile"); !ok {
			return
		} else {
			applicationVersion.ValuesFileData = data
			if _, err := applicationVersion.ParsedValuesFile(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if data, ok := readMultipartFile(w, r, "templatefile"); !ok {
			return
		} else {
			// Template file is taken without parsing on purpose.
			// Some uses might use a non-yaml template here.
			applicationVersion.TemplateFileData = data
		}
	}

	if err := applicationVersion.Validate(application.Type); err != nil {
		log.Error("invalid application version", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := types.ValidateVersionsForStrategy(
		application.VersioningStrategy,
		append(slices.Clone(application.Versions), applicationVersion),
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	auth := auth.Authentication.Require(ctx)
	applicationVersion.CreatedByUserAccountID = new(auth.CurrentUserID())
	resources := applicationVersion.Resources
	if err := db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.CreateApplicationVersion(ctx, &applicationVersion); err != nil {
			return err
		}
		if err := db.CreateApplicationVersionResources(ctx, applicationVersion.ID, resources); err != nil {
			return err
		}
		return triggerAutomaticApplicationUpdates(
			ctx, auth.CurrentOrg(), application.ID, new(auth.CurrentUserID()))
	}); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if errors.Is(err, apierrors.ErrAlreadyExists) {
			http.Error(w, "Application version cannot be created because a version with this name already exists.",
				http.StatusBadRequest)
		} else {
			log.Warn("could not create applicationversion", zap.Error(err))
			sentry.GetHubFromContext(r.Context()).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	} else {
		notification.Dispatch(ctx, func(ctx context.Context) error {
			return notification.SendApplicationUpdateAvailableNotifications(ctx, applicationVersion)
		})

		RespondJSON(w, applicationVersionMapper(ctx)(applicationVersion))
	}
}

func updateApplicationVersion(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	existing := internalctx.GetApplication(ctx)
	if request, resp := decodeJSON[api.UpdateApplicationVersionRequest](r); resp != nil {
		return resp, nil
	} else if request.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else if applicationVersionIdFromUrl, err := uuid.Parse(r.PathValue("applicationVersionId")); err != nil {
		return applicationVersionNotFound, nil
	} else if existingIndex := slices.IndexFunc(existing.Versions, func(v types.ApplicationVersion) bool {
		return v.ID == applicationVersionIdFromUrl
	}); existingIndex < 0 {
		return applicationVersionNotFound, nil
	} else {
		updatedVersion := mapping.UpdateApplicationVersionToInternal(request, existing.Versions[existingIndex])
		updated := *existing
		updated.Versions = slices.Clone(existing.Versions)
		updated.Versions[existingIndex] = updatedVersion
		if err := types.ValidateVersionsForStrategy(updated.VersioningStrategy, updated.Versions); err != nil {
			return respond.BadRequest(err.Error()), nil
		} else if err := db.RunTx(ctx, func(ctx context.Context) error {
			if err := db.UpdateApplicationVersion(ctx, &updatedVersion); err != nil {
				return err
			}
			return triggerAutomaticApplicationUpdates(
				ctx, auth.CurrentOrg(), existing.ID, new(auth.CurrentUserID()))
		}); errors.Is(err, apierrors.ErrAlreadyExists) {
			return respond.BadRequest(
				"Application version cannot be updated because a version with this name already exists.",
			), nil
		} else if err != nil {
			return nil, fmt.Errorf("could not update applicationversion: %w", err)
		} else {
			return respond.JSON(applicationVersionMapper(ctx)(updatedVersion)), nil
		}
	}
}

var (
	getApplicationVersionComposeFile = getApplicationVersionFileHandler(func(av types.ApplicationVersion) []byte {
		return av.ComposeFileData
	})

	getApplicationVersionValuesFile = getApplicationVersionFileHandler(func(av types.ApplicationVersion) []byte {
		return av.ValuesFileData
	})
	getApplicationVersionTemplateFile = getApplicationVersionFileHandler(func(av types.ApplicationVersion) []byte {
		return av.TemplateFileData
	})
)

func getApplicationVersionFileHandler(fileAccessor func(types.ApplicationVersion) []byte) http.HandlerFunc {
	return respond.Handler(func(r *http.Request) (respond.Responder, error) {
		if v, resp, err := getAccessibleApplicationVersion(r); resp != nil || err != nil {
			return resp, err
		} else {
			return respond.Stream("application/yaml", bytes.NewReader(fileAccessor(*v))).
				WithHeader("Cache-Control", "max-age=300, private"), nil
		}
	})
}

func getApplicationVersionResources(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	a := auth.Authentication.Require(ctx)
	if version, resp, err := getAccessibleApplicationVersion(r); resp != nil || err != nil {
		return resp, err
	} else {
		var resources []types.ApplicationVersionResource
		if a.CurrentCustomerOrgID() != nil {
			resources, err = db.GetApplicationVersionResourcesVisibleToCustomers(ctx, version.ID)
		} else {
			resources, err = db.GetApplicationVersionResources(ctx, version.ID)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get application version resources: %w", err)
		} else {
			return respond.JSON(resources), nil
		}
	}
}

func deleteApplication(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	application := internalctx.GetApplication(ctx)
	auth := auth.Authentication.Require(ctx)
	if application.OrganizationID != *auth.CurrentOrgID() {
		return applicationNotFound, nil
	} else if err := db.DeleteApplicationWithID(ctx, application.ID); err != nil {
		if errors.Is(err, apierrors.ErrConflict) {
			return respond.BadRequest("could not delete Application because it is still in use"), nil
		} else {
			return nil, fmt.Errorf("error deleting application: %w", err)
		}
	} else {
		return respond.NoContent(), nil
	}
}

var patchImageApplication = patchImageHandler(func(ctx context.Context, body api.PatchImageRequest) (any, error) {
	application := internalctx.GetApplication(ctx)
	if err := db.UpdateApplicationImage(ctx, application, body.ImageID); err != nil {
		return nil, err
	} else {
		return applicationMapper(ctx)(*application), nil
	}
})

func applicationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		applicationID, err := uuid.Parse(r.PathValue("applicationId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		auth := auth.Authentication.Require(ctx)
		application, err := db.GetApplication(ctx, applicationID, *auth.CurrentOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(r.Context()).Error("failed to get application", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			ctx = internalctx.WithApplication(ctx, application)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}
