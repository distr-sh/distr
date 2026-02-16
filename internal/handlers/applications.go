package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ApplicationsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Applications"))
	r.Use(middleware.RequireOrgAndRole)

	r.Get("/", getApplications).
		With(option.Description("List all applications")).
		With(option.Response(http.StatusOK, []api.ApplicationResponse{}))

	r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", createApplication).
		With(option.Description("Create a new application")).
		With(option.Response(http.StatusOK, api.ApplicationResponse{}))

	r.Route("/{applicationId}", func(r chiopenapi.Router) {
		type ApplicationRequest struct {
			ApplicationID string `path:"applicationId"`
		}

		r.With(applicationMiddleware).Group(func(r chiopenapi.Router) {
			r.Get("/", getApplication).
				With(option.Description("Get an application by ID")).
				With(option.Request(ApplicationRequest{})).
				With(option.Response(http.StatusOK, api.ApplicationResponse{}))
			r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Group(func(r chiopenapi.Router) {
					r.Delete("/", deleteApplication).
						With(option.Description("Delete an application")).
						With(option.Request(ApplicationRequest{}))
					r.Put("/", updateApplication).
						With(option.Description("Update an application")).
						With((option.Request(struct {
							ApplicationRequest
							types.Application
						}{}))).
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
						With(middleware.RequireAnyUserRole(types.UserRoleReadWrite, types.UserRoleAdmin)).
						With(middleware.BlockSuperAdmin).
						Post("/", createApplicationVersion).
						With(option.Description("Create a new application version")).
						With(option.Request(struct {
							ApplicationRequest
							types.ApplicationVersion
						}{})).
						With(option.Response(http.StatusOK, types.ApplicationVersion{}))
				})
			r.Route("/{applicationVersionId}", func(r chiopenapi.Router) {
				type ApplicationVersionRequest struct {
					ApplicationRequest
					ApplicationVersionId string `path:"applicationVersionId"`
				}

				r.Get("/", getApplicationVersion).
					With(option.Description("Get an application version")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, types.ApplicationVersion{}))
				r.With(middleware.RequireVendor, middleware.BlockSuperAdmin, applicationMiddleware).
					Put("/", updateApplicationVersion).
					With(option.Description("Update an application version")).
					With(option.Request(struct {
						ApplicationVersionRequest
						types.ApplicationVersion
					}{})).
					With(option.Response(http.StatusOK, types.ApplicationVersion{}))
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
				r.Get("/resources", getApplicationVersionResources).
					With(option.Description("Get application version resources")).
					With(option.Request(ApplicationVersionRequest{})).
					With(option.Response(http.StatusOK, []types.ApplicationVersionResource{}))
			})
		})
	})
}

func createApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	application, err := JsonBody[types.Application](w, r)
	if err != nil {
		return
	} else if application.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err = db.CreateApplication(ctx, &application, *auth.CurrentOrgID()); err != nil {
		log.Warn("could not create application", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err = fmt.Fprintln(w, err); err != nil {
			log.Error("failed to write error to response", zap.Error(err))
		}
	} else if err = json.NewEncoder(w).Encode(application); err != nil {
		log.Error("failed to encode json", zap.Error(err))
	}
}

func updateApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	application, err := JsonBody[types.Application](w, r)
	if err != nil {
		return
	} else if application.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	existing := internalctx.GetApplication(ctx)
	if application.ID == uuid.Nil {
		application.ID = existing.ID
	} else if application.ID != existing.ID || application.Type != existing.Type {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := db.UpdateApplication(ctx, &application, *auth.CurrentOrgID()); err != nil {
		log.Warn("could not update application", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// TODO ?
	// there surely is some way to have the update command returning the versions too, but I don't think it's worth
	// the work right now
	application.Versions = existing.Versions
	RespondJSON(w, mapping.ApplicationToAPI(application))
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

		if err := db.RunTx(ctx, func(ctx context.Context) error {
			appliationNeedsUpdate := false
			if patch.Name != nil && patch.Name != &existing.Name {
				existing.Name = *patch.Name
				appliationNeedsUpdate = true
			}

			if appliationNeedsUpdate {
				if err := db.UpdateApplication(ctx, existing, *auth.CurrentOrgID()); err != nil {
					log.Warn("could not update application", zap.Error(err))
					sentry.GetHubFromContext(ctx).CaptureException(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return err
				}
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
			return nil
		}); err != nil {
			if errors.Is(err, pgx.ErrTxCommitRollback) {
				log.Warn("could not commit db transaction", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		RespondJSON(w, mapping.ApplicationToAPI(*existing))
	}
}

func getApplications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	log := internalctx.GetLogger(ctx)

	org := auth.CurrentOrg()
	var err error
	var applications []types.Application
	if org.HasFeature(types.FeatureLicensing) && auth.CurrentCustomerOrgID() != nil {
		// Get applications based on license owner ID only if there is at least one license in the parent organization
		if licenses, err1 := db.GetApplicationLicensesWithOrganizationID(ctx, *auth.CurrentOrgID(), nil); err1 != nil {
			log.Error("failed to get application licenses", zap.Error(err1))
			sentry.GetHubFromContext(ctx).CaptureException(err1)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if len(licenses) > 0 {
			applications, err = db.GetApplicationsWithLicenseOwnerID(ctx, *auth.CurrentCustomerOrgID())
		} else {
			applications, err = db.GetApplicationsByOrgID(ctx, *auth.CurrentOrgID())
		}
	} else {
		applications, err = db.GetApplicationsByOrgID(ctx, *auth.CurrentOrgID())
	}

	if err != nil {
		log.Error("failed to get applications", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.List(applications, mapping.ApplicationToAPI))
	}
}

func getApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	log := internalctx.GetLogger(ctx)

	org := auth.CurrentOrg()
	if org.HasFeature(types.FeatureLicensing) && auth.CurrentCustomerOrgID() != nil {
		if applicationID, err := uuid.Parse(r.PathValue("applicationId")); err != nil {
			http.NotFound(w, r)
			return
		} else if licenses, err := db.GetApplicationLicensesWithOrganizationID(ctx, *auth.CurrentOrgID(), nil); err != nil {
			log.Error("failed to get application licenses", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if len(licenses) > 0 {
			application, err := db.GetApplicationWithLicenseOwnerID(ctx, *auth.CurrentCustomerOrgID(), applicationID)
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else if err != nil {
				log.Error("failed to get application", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			} else {
				RespondJSON(w, mapping.ApplicationToAPI(*application))
			}
		} else {
			RespondJSON(w, mapping.ApplicationToAPI(*internalctx.GetApplication(ctx)))
		}
	} else {
		RespondJSON(w, mapping.ApplicationToAPI(*internalctx.GetApplication(ctx)))
	}
}

func getApplicationVersion(w http.ResponseWriter, r *http.Request) {
	applicationVersionID, err := uuid.Parse(r.PathValue("applicationVersionId"))
	if err != nil {
		http.NotFound(w, r)
	} else if applicationVersion, err := db.GetApplicationVersion(r.Context(), applicationVersionID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "something went wrong", http.StatusInternalServerError)
		}
	} else {
		RespondJSON(w, applicationVersion)
	}
}

func createApplicationVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	body := r.FormValue("applicationversion")
	var applicationVersion types.ApplicationVersion
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&applicationVersion); err != nil {
		log.Error("failed to deocde version", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	application := internalctx.GetApplication(ctx)
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

	resources := applicationVersion.Resources
	if err := db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.CreateApplicationVersion(ctx, &applicationVersion); err != nil {
			return err
		}
		if err := db.CreateApplicationVersionResources(ctx, applicationVersion.ID, resources); err != nil {
			return err
		}
		return nil
	}); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if errors.Is(err, apierrors.ErrAlreadyExists) {
			http.Error(w, "application version can not be created. Does a version with the same name already exist?",
				http.StatusBadRequest)
		} else {
			log.Warn("could not create applicationversion", zap.Error(err))
			sentry.GetHubFromContext(r.Context()).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	} else {
		RespondJSON(w, applicationVersion)
	}
}

func updateApplicationVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	applicationVersion, err := JsonBody[types.ApplicationVersion](w, r)
	if err != nil {
		return
	}

	applicationVersionIdFromUrl, err := uuid.Parse(r.PathValue("applicationVersionId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	existing := internalctx.GetApplication(ctx)
	var existingVersion *types.ApplicationVersion
	for _, version := range existing.Versions {
		if version.ID == applicationVersionIdFromUrl {
			existingVersion = &version
		}
	}
	if existingVersion == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if applicationVersion.ID == uuid.Nil {
		applicationVersion.ID = existingVersion.ID
	}

	if err := db.UpdateApplicationVersion(ctx, &applicationVersion); err != nil {
		log.Warn("could not update applicationversion", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		RespondJSON(w, applicationVersion)
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
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		applicationVersionID, err := uuid.Parse(r.PathValue("applicationVersionId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if v, err := db.GetApplicationVersion(ctx, applicationVersionID); errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else if err != nil {
			log.Error("failed to get ApplicationVersion from DB", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			data := fileAccessor(*v)
			w.Header().Add("Content-Type", "application/yaml")
			w.Header().Add("Cache-Control", "max-age=300, private")
			if data != nil {
				if _, err := w.Write(data); err != nil {
					log.Warn("failed to write file to response", zap.Error(err))
				}
			}
		}
	}
}

func getApplicationVersionResources(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	a := auth.Authentication.Require(ctx)
	applicationVersionID, err := uuid.Parse(r.PathValue("applicationVersionId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var resources []types.ApplicationVersionResource
	if a.CurrentCustomerOrgID() != nil {
		resources, err = db.GetApplicationVersionResourcesVisibleToCustomers(ctx, applicationVersionID)
	} else {
		resources, err = db.GetApplicationVersionResources(ctx, applicationVersionID)
	}
	if err != nil {
		log.Error("failed to get application version resources", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, resources)
}

func deleteApplication(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	application := internalctx.GetApplication(ctx)
	auth := auth.Authentication.Require(ctx)
	if application.OrganizationID != *auth.CurrentOrgID() {
		http.NotFound(w, r)
	} else if err := db.DeleteApplicationWithID(ctx, application.ID); err != nil {
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "could not delete Application because it is still in use", http.StatusBadRequest)
		} else {
			log.Warn("error deleting application", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

var patchImageApplication = patchImageHandler(func(ctx context.Context, body api.PatchImageRequest) (any, error) {
	application := internalctx.GetApplication(ctx)
	if err := db.UpdateApplicationImage(ctx, application, body.ImageID); err != nil {
		return nil, err
	} else {
		return mapping.ApplicationToAPI(*application), nil
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
