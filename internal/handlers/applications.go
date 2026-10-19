package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/notification"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
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

		r.With(respond.Middleware(applicationMiddleware)).Group(func(r chiopenapi.Router) {
			r.Get("/", respond.Handler(getApplication)).
				With(option.Description("Get an application by ID")).
				With(option.Request(ApplicationRequest{})).
				With(option.Response(http.StatusOK, api.ApplicationResponse{}))
			r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Group(func(r chiopenapi.Router) {
					r.Delete("/", respond.Handler(deleteApplication)).
						With(option.Description("Delete an application")).
						With(option.Request(ApplicationRequest{}))
					r.Put("/", respond.Handler(updateApplication)).
						With(option.Description("Update an application")).
						With(option.Request(struct {
							ApplicationRequest
							api.UpdateApplicationRequest
						}{})).
						With(option.Response(http.StatusOK, api.ApplicationResponse{}))
					r.Patch("/", respond.Handler(patchApplication)).
						With(option.Description("Partially update an application")).
						With(option.Request(struct {
							ApplicationRequest
							api.PatchApplicationRequest
						}{})).
						With(option.Response(http.StatusOK, api.ApplicationResponse{}))
					r.Patch("/image", respond.Handler(patchImageApplication)).
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
			r.With(respond.Middleware(applicationMiddleware)).
				Group(func(r chiopenapi.Router) {
					r.With(middleware.RequireVendor).
						With(middleware.RequireReadWriteOrAdmin).
						With(middleware.BlockSuperAdmin).
						Post("/", respond.Handler(createApplicationVersion)).
						With(option.Description("Create a new application version from a multipart form. " +
							"The applicationversion field holds the version as JSON, for example {\"name\":\"1.0.0\"}. " +
							"Helm versions also need chartType, chartUrl and chartVersion, and chartName for a " +
							"repository chart. Each file part is limited to 100 KiB and needs a YAML or JSON content type, " +
							"for example curl -F 'composefile=@compose.yaml;type=application/yaml'.")).
						With(option.Request(struct {
							ApplicationRequest
							api.CreateApplicationVersionFormRequest
						}{}, option.ContentType("multipart/form-data"))).
						With(option.Response(http.StatusOK, api.ApplicationVersionResponse{}))
				})
			r.Route("/{applicationVersionId}", func(r chiopenapi.Router) {
				type ApplicationVersionRequest struct {
					ApplicationRequest
					ApplicationVersionId string `path:"applicationVersionId"`
				}

				r.With(respond.Middleware(applicationMiddleware)).
					Get("/", respond.Handler(getApplicationVersion)).
					With(option.Description("Get an application version")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, api.ApplicationVersionResponse{}))
				r.With(middleware.RequireVendor).
					With(middleware.RequireReadWriteOrAdmin).
					With(middleware.BlockSuperAdmin).
					With(respond.Middleware(applicationMiddleware)).
					Put("/", respond.Handler(updateApplicationVersion)).
					With(option.Description("Update an application version")).
					With(option.Request(struct {
						ApplicationVersionRequest
						api.UpdateApplicationVersionRequest
					}{})).
					With(option.Response(http.StatusOK, api.ApplicationVersionResponse{}))
				r.Get("/compose-file", respond.Handler(getApplicationVersionComposeFile)).
					With(option.Description("Get application version compose file")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, map[string]any{}, option.ContentType("application/yaml")))
				r.Get("/template-file", respond.Handler(getApplicationVersionTemplateFile)).
					With(option.Description("Get application version template file")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
				r.Get("/values-file", respond.Handler(getApplicationVersionValuesFile)).
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

func createApplication(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if request, resp := decodeJSON[api.CreateApplicationRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
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

func updateApplication(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	existing := internalctx.GetApplication(ctx)
	if request, resp := decodeJSON[api.UpdateApplicationRequest](r); resp != nil {
		return resp, nil
	} else if request.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else {
		application := mapping.UpdateApplicationToInternal(request, *existing)
		// A client written before the strategy existed does not send it, and must not silently move
		// the application off the one it has.
		if application.VersioningStrategy == "" {
			application.VersioningStrategy = existing.VersioningStrategy
		}
		if err := checkApplicationSettings(auth.CurrentOrg(), &application, existing.VersioningStrategy); err != nil {
			return respond.BadRequest(err.Error()), nil
		} else if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
			if err := db.UpdateApplication(ctx, &application, *auth.CurrentOrgID()); err != nil {
				return nil, fmt.Errorf("could not update application: %w", err)
			}
			// TODO ?
			// there surely is some way to have the update command returning the versions too, but I don't think it's worth
			// the work right now
			application.Versions = existing.Versions

			if err := triggerAutomaticApplicationUpdates(
				ctx, auth.CurrentOrg(), application.ID, new(auth.CurrentUserID()),
			); err != nil {
				return nil, fmt.Errorf("failed to trigger automatic application updates: %w", err)
			}
			return nil, nil
		}); resp != nil || err != nil {
			return resp, err
		} else {
			return respond.JSON(applicationMapper(ctx)(application)), nil
		}
	}
}

func patchApplication(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	existing := internalctx.GetApplication(ctx)
	patch, resp := decodeJSON[api.PatchApplicationRequest](r)
	if resp != nil {
		return resp, nil
	}

	previousStrategy := existing.VersioningStrategy
	patched := *existing
	if patch.Name != nil {
		if *patch.Name == "" {
			return respond.BadRequest("name is required"), nil
		}
		patched.Name = *patch.Name
	}
	if patch.VersioningStrategy != nil {
		patched.VersioningStrategy = *patch.VersioningStrategy
	}
	if patch.AllowAutomaticUpdates != nil {
		patched.AllowAutomaticUpdates = *patch.AllowAutomaticUpdates
	}
	if err := checkApplicationSettings(auth.CurrentOrg(), &patched, previousStrategy); err != nil {
		return respond.BadRequest(err.Error()), nil
	}
	applicationNeedsUpdate := patched.Name != existing.Name ||
		patched.VersioningStrategy != existing.VersioningStrategy ||
		patched.AllowAutomaticUpdates != existing.AllowAutomaticUpdates

	if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		if applicationNeedsUpdate {
			versions := existing.Versions
			if err := db.UpdateApplication(ctx, &patched, *auth.CurrentOrgID()); err != nil {
				return nil, fmt.Errorf("could not update application: %w", err)
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
				return respond.BadRequest(fmt.Sprintf("no application version found with ID %v", vp.ID)), nil
			}

			versionNeedsUpdate := false
			if !util.PtrEq(ev.ArchivedAt, vp.ArchivedAt) {
				ev.ArchivedAt = vp.ArchivedAt
				versionNeedsUpdate = true
			}

			if versionNeedsUpdate {
				if err := db.UpdateApplicationVersion(ctx, ev); err != nil {
					return nil, fmt.Errorf("could not update application version: %w", err)
				}
			}
		}

		// Un-archiving a version, or switching the strategy, can make another version the
		// newest one.
		if err := triggerAutomaticApplicationUpdates(
			ctx, auth.CurrentOrg(), existing.ID, new(auth.CurrentUserID()),
		); err != nil {
			return nil, fmt.Errorf("failed to trigger automatic application updates: %w", err)
		}
		return nil, nil
	}); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.JSON(applicationMapper(ctx)(*existing)), nil
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
			if errors.Is(err, distrerr.ErrNotFound) {
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
		); errors.Is(err, distrerr.ErrNotFound) {
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

func createApplicationVersion(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	application := internalctx.GetApplication(ctx)

	request, resp := decodeJSONFormValue[api.CreateApplicationVersionRequest](r, "applicationversion")
	if resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	}

	applicationVersion := mapping.CreateApplicationVersionToInternal(request)
	applicationVersion.ApplicationID = application.ID
	applicationVersion.CreatedByUserAccountID = new(auth.CurrentUserID())
	resources := applicationVersion.Resources

	if resp, err := readTemplateFile(r, &applicationVersion); resp != nil || err != nil {
		return resp, err
	} else if resp, err := readDeploymentFile(r, application.Type, &applicationVersion); resp != nil || err != nil {
		return resp, err
	} else if err := applicationVersion.Validate(application.Type); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if err := types.ValidateVersionsForStrategy(
		application.VersioningStrategy,
		append(slices.Clone(application.Versions), applicationVersion),
	); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if err := db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.CreateApplicationVersion(ctx, &applicationVersion); err != nil {
			return err
		}
		if err := db.CreateApplicationVersionResources(ctx, applicationVersion.ID, resources); err != nil {
			return err
		}
		return triggerAutomaticApplicationUpdates(
			ctx, auth.CurrentOrg(), application.ID, new(auth.CurrentUserID()))
	}); errors.Is(err, distrerr.ErrNotFound) {
		return applicationNotFound, nil
	} else if errors.Is(err, distrerr.ErrAlreadyExists) {
		return respond.BadRequest(
			"Application version cannot be created because a version with this name already exists.",
		), nil
	} else if err != nil {
		return nil, fmt.Errorf("could not create applicationversion: %w", err)
	} else {
		notification.Dispatch(ctx, func(ctx context.Context) error {
			return notification.SendApplicationUpdateAvailableNotifications(ctx, applicationVersion)
		})

		return respond.JSON(applicationVersionMapper(ctx)(applicationVersion)), nil
	}
}

// readTemplateFile takes the template file without parsing on purpose, since some use a non-YAML template.
func readTemplateFile(r *http.Request, version *types.ApplicationVersion) (respond.Responder, error) {
	data, resp, err := readMultipartFile(r, "templatefile")
	version.TemplateFileData = data
	return resp, err
}

// readDeploymentFile reads the compose file of a Docker application or the values file of a Helm one.
func readDeploymentFile(
	r *http.Request,
	deploymentType types.DeploymentType,
	version *types.ApplicationVersion,
) (respond.Responder, error) {
	if deploymentType == types.DeploymentTypeDocker {
		data, resp, err := readMultipartFile(r, "composefile")
		if resp != nil || err != nil {
			return resp, err
		}
		version.ComposeFileData = data
		if _, err := version.ParsedComposeFile(); err != nil {
			return respond.BadRequest(err.Error()), nil
		}
	} else {
		data, resp, err := readMultipartFile(r, "valuesfile")
		if resp != nil || err != nil {
			return resp, err
		}
		version.ValuesFileData = data
		if _, err := version.ParsedValuesFile(); err != nil {
			return respond.BadRequest(err.Error()), nil
		}
	}
	return nil, nil
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
		}); errors.Is(err, distrerr.ErrAlreadyExists) {
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

func getApplicationVersionComposeFile(r *http.Request) (respond.Responder, error) {
	return applicationVersionFile(r, func(av types.ApplicationVersion) []byte { return av.ComposeFileData })
}

func getApplicationVersionValuesFile(r *http.Request) (respond.Responder, error) {
	return applicationVersionFile(r, func(av types.ApplicationVersion) []byte { return av.ValuesFileData })
}

func getApplicationVersionTemplateFile(r *http.Request) (respond.Responder, error) {
	return applicationVersionFile(r, func(av types.ApplicationVersion) []byte { return av.TemplateFileData })
}

func applicationVersionFile(r *http.Request, file func(types.ApplicationVersion) []byte) (respond.Responder, error) {
	if v, resp, err := getAccessibleApplicationVersion(r); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.Stream("application/yaml", bytes.NewReader(file(*v))).
			WithHeader("Cache-Control", "max-age=300, private"), nil
	}
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
		if errors.Is(err, distrerr.ErrConflict) {
			return respond.BadRequest("could not delete application because it is still in use"), nil
		} else {
			return nil, fmt.Errorf("error deleting application: %w", err)
		}
	} else {
		return respond.NoContent(), nil
	}
}

func patchImageApplication(r *http.Request) (respond.Responder, error) {
	return patchImage(r, func(ctx context.Context, body api.PatchImageRequest) (any, error) {
		application := internalctx.GetApplication(ctx)
		if err := db.UpdateApplicationImage(ctx, application, body.ImageID); err != nil {
			return nil, err
		} else {
			return applicationMapper(ctx)(*application), nil
		}
	})
}

func applicationMiddleware(r *http.Request) (*http.Request, respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if applicationID, err := uuid.Parse(r.PathValue("applicationId")); err != nil {
		return nil, applicationNotFound, nil
	} else if application, err := db.GetApplication(
		ctx, applicationID, *auth.CurrentOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return nil, applicationNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get application: %w", err)
	} else {
		return r.WithContext(internalctx.WithApplication(ctx, application)), nil, nil
	}
}
