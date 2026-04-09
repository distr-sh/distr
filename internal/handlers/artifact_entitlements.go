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

func ArtifactEntitlementsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Artifacts", "Licensing"))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireVendor, middleware.LicensingFeatureFlagEnabledMiddleware)
	r.Get("/", getArtifactEntitlements).
		With(option.Description("List all artifact entitlements")).
		With(option.Response(http.StatusOK, []types.ArtifactEntitlement{}))
	r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Post("/", createArtifactEntitlement).
			With(option.Description("Create a new artifact entitlement")).
			With(option.Request(types.ArtifactEntitlement{})).
			With(option.Response(http.StatusOK, types.ArtifactEntitlement{}))
		r.With(artifactEntitlementMiddleware).Route("/{artifactEntitlementId}", func(r chiopenapi.Router) {
			type ArtifactEntitlementRequest struct {
				ArtifactEntitlementID uuid.UUID `path:"artifactEntitlementId"`
			}

			r.Put("/", updateArtifactEntitlement).
				With(option.Description("Update an artifact entitlement")).
				With(option.Request(struct {
					ArtifactEntitlementRequest
					types.ArtifactEntitlement
				}{})).
				With(option.Response(http.StatusOK, types.ArtifactEntitlement{}))
			r.Delete("/", deleteArtifactEntitlement).
				With(option.Description("Delete an artifact entitlement")).
				With(option.Request(ArtifactEntitlementRequest{}))
		})
	})
}

func getArtifactEntitlements(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	if entitlements, err := db.GetArtifactEntitlements(ctx, *auth.CurrentOrgID()); err != nil {
		log.Error("failed to get artifact entitlements", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	} else {
		RespondJSON(w, entitlements)
	}
}

func createArtifactEntitlement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	entitlement, err := JsonBody[types.ArtifactEntitlement](w, r)
	if err != nil {
		return
	}
	entitlement.OrganizationID = *auth.CurrentOrgID()

	if err = validateEntitlementSelections(entitlement); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		err := db.CreateArtifactEntitlement(ctx, &entitlement.ArtifactEntitlementBase)
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "An artifact entitlement with this name already exists", http.StatusBadRequest)
			return err
		} else if err != nil {
			log.Warn("could not create artifact entitlement", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if err := addArtifacts(ctx, entitlement, log, w); err != nil {
			return err
		}

		RespondJSON(w, entitlement)
		return nil
	})
}

func updateArtifactEntitlement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	entitlement, err := JsonBody[types.ArtifactEntitlement](w, r)
	if err != nil {
		return
	}
	entitlement.OrganizationID = *auth.CurrentOrgID()

	if err = validateEntitlementSelections(entitlement); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existing := internalctx.GetArtifactEntitlement(ctx)
	if entitlement.ID == uuid.Nil {
		entitlement.ID = existing.ID
	} else if entitlement.ID != existing.ID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		err := db.UpdateArtifactEntitlement(ctx, &entitlement.ArtifactEntitlementBase)
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "An artifact entitlement with this name already exists", http.StatusBadRequest)
			return err
		} else if err != nil {
			log.Warn("could not update artifact entitlement", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if err := db.RemoveAllArtifactsFromEntitlement(ctx, entitlement.ID); err != nil {
			log.Warn("could not update artifact entitlement selection", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if err := addArtifacts(ctx, entitlement, log, w); err != nil {
			return err
		}

		RespondJSON(w, entitlement)
		return nil
	})
}

func validateEntitlementSelections(entitlement types.ArtifactEntitlement) error {
	artifactIdSet := make(map[uuid.UUID]struct{})
	for _, selection := range entitlement.Artifacts {
		if _, exists := artifactIdSet[selection.ArtifactID]; exists {
			return errors.New("cannot select same artifact multiple times")
		}
		versionIdSet := make(map[uuid.UUID]struct{})
		for _, version := range selection.VersionIDs {
			if _, exists := versionIdSet[version]; exists {
				return errors.New("cannot select same version of artifact multiple times")
			}
			versionIdSet[version] = struct{}{}
		}
		artifactIdSet[selection.ArtifactID] = struct{}{}
	}
	return nil
}

func addArtifacts(
	ctx context.Context, entitlement types.ArtifactEntitlement, log *zap.Logger, w http.ResponseWriter,
) error {
	for _, selection := range entitlement.Artifacts {
		if len(selection.VersionIDs) == 0 {
			if err := db.AddArtifactToArtifactEntitlement(ctx, entitlement.ID, selection.ArtifactID, nil); err != nil {
				log.Warn("could not add version to entitlement", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		}
		for _, versionID := range selection.VersionIDs {
			if err := db.AddArtifactToArtifactEntitlement(ctx, entitlement.ID, selection.ArtifactID, &versionID); err != nil {
				log.Warn("could not add version to entitlement", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		}
	}
	return nil
}

func deleteArtifactEntitlement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	entitlement := internalctx.GetArtifactEntitlement(ctx)
	auth := auth.Authentication.Require(ctx)
	if entitlement.OrganizationID != *auth.CurrentOrgID() {
		http.NotFound(w, r)
	} else if err := db.DeleteArtifactEntitlementWithID(ctx, entitlement.ID); errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "could not delete entitlement because it is still in use", http.StatusBadRequest)
	} else if err != nil {
		log.Warn("error deleting entitlement", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func artifactEntitlementMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		a := auth.Authentication.Require(ctx)
		if entitlementId, err := uuid.Parse(r.PathValue("artifactEntitlementId")); err != nil {
			http.Error(w, "artifactEntitlementId is not a valid UUID", http.StatusBadRequest)
		} else if entitlement, err := db.GetArtifactEntitlementByID(ctx, entitlementId); errors.Is(
			err, apierrors.ErrNotFound,
		) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(r.Context()).Error("failed to get entitlement", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else if entitlement.OrganizationID != *a.CurrentOrgID() {
			w.WriteHeader(http.StatusNotFound)
		} else {
			ctx = internalctx.WithArtifactEntitlement(ctx, entitlement)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}
