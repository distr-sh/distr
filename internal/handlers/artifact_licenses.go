package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
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

func ArtifactLicensesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Artifacts", "Licensing"))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireVendor, middleware.LicensingFeatureFlagEnabledMiddleware)
	r.Get("/", getArtifactLicenses).
		With(option.Description("List all artifact licenses")).
		With(option.Response(http.StatusOK, []types.ArtifactLicense{}))
	r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Post("/", createArtifactLicense).
			With(option.Description("Create a new artifact license")).
			With(option.Request(types.ArtifactLicense{})).
			With(option.Response(http.StatusOK, types.ArtifactLicense{}))
		r.With(artifactLicenseMiddleware).Route("/{artifactLicenseId}", func(r chiopenapi.Router) {
			type ArtifactLicenseRequest struct {
				ArtifactLicenseID uuid.UUID `path:"artifactLicenseId"`
			}

			r.Put("/", updateArtifactLicense).
				With(option.Description("Update an artifact license")).
				With(option.Request(struct {
					ArtifactLicenseRequest
					types.ArtifactLicense
				}{})).
				With(option.Response(http.StatusOK, types.ArtifactLicense{}))
			r.Delete("/", deleteArtifactLicense).
				With(option.Description("Delete an artifact license")).
				With(option.Request(ArtifactLicenseRequest{}))
		})
	})
}

func getArtifactLicenses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	if licenses, err := db.GetArtifactLicenses(ctx, *auth.CurrentOrgID()); err != nil {
		log.Error("failed to get artifact licenses", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	} else {
		RespondJSON(w, licenses)
	}
}

func createArtifactLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	license, err := JsonBody[types.ArtifactLicense](w, r)
	if err != nil {
		return
	}
	license.OrganizationID = *auth.CurrentOrgID()

	if err = validateLicenseSelections(license); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.CreateArtifactLicense(ctx, &license.ArtifactLicenseBase); errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "An artifact license with this name already exists", http.StatusBadRequest)
			return err
		} else if err != nil {
			log.Warn("could not create artifact license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if err := addArtifacts(ctx, license, log, w); err != nil {
			return err
		}

		RespondJSON(w, license)
		return nil
	})
}

func updateArtifactLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	license, err := JsonBody[types.ArtifactLicense](w, r)
	if err != nil {
		return
	}
	license.OrganizationID = *auth.CurrentOrgID()

	if err = validateLicenseSelections(license); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existing := internalctx.GetArtifactLicense(ctx)
	if license.ID == uuid.Nil {
		license.ID = existing.ID
	} else if license.ID != existing.ID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.UpdateArtifactLicense(ctx, &license.ArtifactLicenseBase); errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "An artifact license with this name already exists", http.StatusBadRequest)
			return err
		} else if err != nil {
			log.Warn("could not update artifact license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if err := db.RemoveAllArtifactsFromLicense(ctx, license.ID); err != nil {
			log.Warn("could not update artifct license selection", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if err := addArtifacts(ctx, license, log, w); err != nil {
			return err
		}

		RespondJSON(w, license)
		return nil
	})
}

func validateLicenseSelections(license types.ArtifactLicense) error {
	artifactIdSet := make(map[uuid.UUID]struct{})
	for _, selection := range license.Artifacts {
		if _, exists := artifactIdSet[selection.ArtifactID]; exists {
			return errors.New("cannot select same artifact multiple times")
		}
		versionIdSet := make(map[uuid.UUID]struct{})
		for _, version := range selection.VersionIDs {
			if _, exists := versionIdSet[selection.ArtifactID]; exists {
				return errors.New("cannot select same version of artifact multiple times")
			}
			versionIdSet[version] = struct{}{}
		}
		artifactIdSet[selection.ArtifactID] = struct{}{}
	}
	return nil
}

func addArtifacts(ctx context.Context, license types.ArtifactLicense, log *zap.Logger, w http.ResponseWriter) error {
	for _, selection := range license.Artifacts {
		if len(selection.VersionIDs) == 0 {
			if err := db.AddArtifactToArtifactLicense(ctx, license.ID, selection.ArtifactID, nil); err != nil {
				log.Warn("could not add version to license", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		}
		for _, versionID := range selection.VersionIDs {
			if err := db.AddArtifactToArtifactLicense(ctx, license.ID, selection.ArtifactID, &versionID); err != nil {
				log.Warn("could not add version to license", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		}
	}
	return nil
}

func deleteArtifactLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	license := internalctx.GetArtifactLicense(ctx)
	auth := auth.Authentication.Require(ctx)
	if license.OrganizationID != *auth.CurrentOrgID() {
		http.NotFound(w, r)
	} else if err := db.DeleteArtifactLicenseWithID(ctx, license.ID); errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "could not delete license because it is still in use", http.StatusBadRequest)
	} else if err != nil {
		log.Warn("error deleting license", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func artifactLicenseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if licenseId, err := uuid.Parse(r.PathValue("artifactLicenseId")); err != nil {
			http.Error(w, "artifactLicenseId is not a valid UUID", http.StatusBadRequest)
		} else if license, err := db.GetArtifactLicenseByID(ctx, licenseId); errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(r.Context()).Error("failed to get license", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			ctx = internalctx.WithArtifactLicense(ctx, license)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}
