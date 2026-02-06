package handlers

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ApplicationLicensesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Applications", "Licensing"))
	r.Use(middleware.RequireOrgAndRole, middleware.LicensingFeatureFlagEnabledMiddleware)
	r.Get("/", getApplicationLicenses).
		With(option.Description("List all application licenses")).
		With(option.Response(http.StatusOK, []types.ApplicationLicense{}))
	r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", createApplicationLicense).
		With(option.Description("Create a new application license")).
		With(option.Request(types.ApplicationLicenseWithVersions{}))
	r.With(applicationLicenseMiddleware).Route("/{applicationLicenseId}", func(r chiopenapi.Router) {
		type ApplicationLicenseRequest struct {
			ApplicationLicenseId string `path:"applicationLicenseId"`
		}

		r.Get("/", getApplicationLicense).
			With(option.Description("Get an application license")).
			With(option.Request(ApplicationLicenseRequest{})).
			With(option.Response(http.StatusOK, types.ApplicationLicense{}))
		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Group(func(r chiopenapi.Router) {
				r.Delete("/", deleteApplicationLicense).
					With(option.Description("Delete an application license")).
					With(option.Request(ApplicationLicenseRequest{}))
				r.Put("/", updateApplicationLicense).
					With(option.Description("Update an application license")).
					With(option.Request(struct {
						ApplicationLicenseRequest
						types.ApplicationLicenseWithVersions
					}{})).
					With(option.Response(http.StatusOK, types.ApplicationLicense{}))
			})
	})
}

func createApplicationLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	license, err := JsonBody[types.ApplicationLicenseWithVersions](w, r)
	if err != nil {
		return
	}
	license.OrganizationID = *auth.CurrentOrgID()

	sanitizeRegistryInput(license)

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.CreateApplicationLicense(ctx, &license.ApplicationLicenseBase); errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "A license with this name already exists", http.StatusBadRequest)
			return err
		} else if err != nil {
			log.Warn("could not create license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		for _, version := range license.Versions {
			if err := db.AddVersionToApplicationLicense(ctx, &license.ApplicationLicenseBase, version.ID); err != nil {
				log.Warn("could not add version to license", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		}

		if createdLicense, err := db.GetApplicationLicenseByID(ctx, license.ID); err != nil {
			log.Warn("could not read previously created license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			return err
		} else {
			RespondJSON(w, createdLicense)
		}

		return nil
	})
}

func updateApplicationLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	license, err := JsonBody[types.ApplicationLicenseWithVersions](w, r)
	if err != nil {
		return
	}
	license.OrganizationID = *auth.CurrentOrgID()

	existing := internalctx.GetApplicationLicense(ctx)
	if license.ID == uuid.Nil {
		license.ID = existing.ID
	} else if license.ID != existing.ID {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if existing.CustomerOrganizationID != nil &&
		(license.CustomerOrganizationID == nil || *existing.CustomerOrganizationID != *license.CustomerOrganizationID) {
		http.Error(w, "Changing the license owner is not allowed", http.StatusBadRequest)
		return
	} else if existing.ApplicationID != license.ApplicationID {
		http.Error(w, "Changing the application is not allowed", http.StatusBadRequest)
		return
	}
	sanitizeRegistryInput(license)

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.UpdateApplicationLicense(ctx, &license.ApplicationLicenseBase); errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "A license with this name already exists", http.StatusBadRequest)
			return err
		} else if err != nil {
			log.Warn("could not update license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return err
		}

		for _, version := range license.Versions {
			alreadyExists := slices.ContainsFunc(existing.Versions, func(v types.ApplicationVersion) bool {
				return v.ID == version.ID
			})
			if !alreadyExists {
				if len(existing.Versions) == 0 {
					// we don't allow narrowing down the scope yet. If the existing license allows all versions,
					// setting some specific ones is not possible anymore
					err = errors.New("narrowing down license scope is not allowed yet")
					http.Error(w, err.Error(), http.StatusBadRequest)
					return err
				} else {
					if err := db.AddVersionToApplicationLicense(ctx, &license.ApplicationLicenseBase, version.ID); err != nil {
						log.Warn("could not add version to license", zap.Error(err))
						sentry.GetHubFromContext(ctx).CaptureException(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return err
					}
				}
			}
		}

		for _, existingVersion := range existing.Versions {
			stillExists := slices.ContainsFunc(license.Versions, func(v types.ApplicationVersion) bool {
				return v.ID == existingVersion.ID
			})
			if !stillExists {
				if len(license.Versions) > 0 {
					// for now, removing specific versions from the license is not possible
					// for removal we also would have to check whether this version is used in some deployment target
					err = errors.New("narrowing down license scope is not allowed yet")
					http.Error(w, err.Error(), http.StatusBadRequest)
					return err
				} else {
					// however removing the relations is possible iff the user chose "all versions" by versions = []
					if err := db.RemoveVersionFromApplicationLicense(
						ctx, &license.ApplicationLicenseBase, existingVersion.ID); err != nil {
						log.Warn("could not remove version from license", zap.Error(err))
						sentry.GetHubFromContext(ctx).CaptureException(err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return err
					}
				}
			}
		}

		if updatedLicense, err := db.GetApplicationLicenseByID(ctx, license.ID); err != nil {
			log.Warn("could not read previously updated license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			return err
		} else {
			RespondJSON(w, updatedLicense)
		}

		return nil
	})
}

func sanitizeRegistryInput(license types.ApplicationLicenseWithVersions) {
	if license.RegistryURL == nil || (*license.RegistryURL) == "" {
		license.RegistryURL = nil
		license.RegistryUsername = nil
		license.RegistryPassword = nil
	}
}

func getApplicationLicenses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	var applicationId *uuid.UUID
	if applicationidParam := r.URL.Query().Get("applicationId"); applicationidParam != "" {
		if id, err := uuid.Parse(applicationidParam); err != nil {
			http.Error(w, "applicationId is not a valid UUID", http.StatusBadRequest)
			return
		} else {
			applicationId = &id
		}
	}
	if auth.CurrentCustomerOrgID() == nil {
		if licenses, err := db.GetApplicationLicensesWithOrganizationID(
			ctx, *auth.CurrentOrgID(), applicationId); err != nil {
			internalctx.GetLogger(ctx).Error("failed to get licenses", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			RespondJSON(w, licenses)
		}
	} else {
		if licenses, err := db.GetApplicationLicensesWithCustomerOrganizationID(
			ctx, *auth.CurrentCustomerOrgID(), *auth.CurrentOrgID(), applicationId,
		); err != nil {
			internalctx.GetLogger(ctx).Error("failed to get licenses", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			RespondJSON(w, licenses)
		}
	}
}

func getApplicationLicense(w http.ResponseWriter, r *http.Request) {
	license := internalctx.GetApplicationLicense(r.Context())
	RespondJSON(w, license)
}

func deleteApplicationLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	license := internalctx.GetApplicationLicense(ctx)
	auth := auth.Authentication.Require(ctx)
	if license.OrganizationID != *auth.CurrentOrgID() {
		http.NotFound(w, r)
	} else if err := db.DeleteApplicationLicenseWithID(ctx, license.ID); errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "could not delete license because it is still in use", http.StatusBadRequest)
	} else if err != nil {
		log.Warn("error deleting license", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func applicationLicenseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)
		if licenseId, err := uuid.Parse(r.PathValue("applicationLicenseId")); err != nil {
			http.Error(w, "applicationLicenseId is not a valid UUID", http.StatusBadRequest)
		} else if license, err := db.GetApplicationLicenseByID(ctx, licenseId); errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else if !canSeeLicense(auth, license) {
			w.WriteHeader(http.StatusForbidden)
		} else {
			ctx = internalctx.WithApplicationLicense(ctx, license)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func canSeeLicense(auth authinfo.AuthInfo, license *types.ApplicationLicense) bool {
	if license.OrganizationID != *auth.CurrentOrgID() {
		return false
	}
	if auth.CurrentCustomerOrgID() != nil {
		if license.CustomerOrganizationID == nil || *license.CustomerOrganizationID != *auth.CurrentCustomerOrgID() {
			return false
		}
	}
	return true
}
