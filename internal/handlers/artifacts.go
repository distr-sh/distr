package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func ArtifactsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Artifacts"))
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", getArtifacts).
		With(option.Description("List all artifacts")).
		With(option.Response(http.StatusOK, []api.ArtifactsResponse{}))
	r.With(artifactMiddleware).Route("/{artifactId}", func(r chiopenapi.Router) {
		type ArtifactRequest struct {
			ArtifactID uuid.UUID `path:"artifactId"`
		}

		r.Get("/", getArtifact).
			With(option.Description("Get an artifact by ID")).
			With(option.Request(ArtifactRequest{})).
			With(option.Response(http.StatusOK, []api.ArtifactResponse{}))
		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Group(func(r chiopenapi.Router) {
				r.Patch("/image", patchImageArtifactHandler).
					With(option.Description("Update artifact image")).
					With(option.Request(struct {
						ArtifactRequest
						api.PatchImageRequest
					}{})).
					With(option.Response(http.StatusOK, []api.ArtifactResponse{}))
				r.Delete("/", deleteArtifactHandler).
					With(option.Description("Delete an artifact")).
					With(option.Request(ArtifactRequest{}))
				r.Delete("/tags/{tagName}", deleteArtifactTagHandler).
					With(option.Description("Delete an artifact tag")).
					With(option.Request(struct {
						ArtifactRequestTagName string `path:"tagName"`
					}{}))
			})
	})
}

func getArtifacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	var artifacts []types.ArtifactWithDownloads
	var err error
	if auth.CurrentOrg().HasFeature(types.FeatureLicensing) && auth.CurrentCustomerOrgID() != nil {
		if licenses, err1 := db.GetArtifactLicenses(ctx, *auth.CurrentOrgID()); err1 != nil {
			log.Error("failed to get artifact licenses", zap.Error(err1))
			sentry.GetHubFromContext(ctx).CaptureException(err1)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else if len(licenses) > 0 {
			artifacts, err = db.GetArtifactsByLicenseOwnerID(ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID())
		} else {
			artifacts, err = db.GetArtifactsByOrgID(ctx, *auth.CurrentOrgID())
		}
	} else {
		artifacts, err = db.GetArtifactsByOrgID(ctx, *auth.CurrentOrgID())
	}

	if err != nil {
		log.Error("failed to get artifacts", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.List(artifacts, mapping.ArtifactsWithDownloadsToAPI))
	}
}

func getArtifact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	RespondJSON(w, mapping.ArtifactToAPI(*internalctx.GetArtifact(ctx)))
}

var patchImageArtifactHandler = patchImageHandler(func(ctx context.Context, body api.PatchImageRequest) (any, error) {
	artifact := internalctx.GetArtifact(ctx)
	if err := db.UpdateArtifactImage(ctx, artifact, body.ImageID); err != nil {
		return nil, err
	} else {
		return mapping.ArtifactToAPI(*artifact), nil
	}
})

func deleteArtifactHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	artifact := internalctx.GetArtifact(ctx)

	err := db.RunTx(ctx, func(ctx context.Context) error {
		isReferenced, err := db.ArtifactIsReferencedInLicenses(ctx, artifact.ID)
		if err != nil {
			return err
		}
		if isReferenced {
			return apierrors.NewBadRequest("Cannot delete artifact: it is referenced in one or more licenses.")
		}

		if err := db.DeleteArtifactWithID(ctx, artifact.ID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, apierrors.ErrBadRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Error("error deleting artifact", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteArtifactTagHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	artifact := internalctx.GetArtifact(ctx)

	tagName := r.PathValue("tagName")
	if tagName == "" {
		http.NotFound(w, r)
		return
	}

	err := db.RunTx(ctx, func(ctx context.Context) error {
		// Step 1: Validate version exists and fetch it
		version, err := db.GetArtifactVersionByTag(ctx, artifact.ID, tagName)
		if err != nil {
			return err
		}

		// Step 2: Fetch all versions with the same digest
		versionsWithSameDigest, err := db.GetArtifactVersionsByDigest(ctx, artifact.ID, string(version.ManifestBlobDigest))
		if err != nil {
			return err
		}

		// Step 3: Enhanced license check
		if err := db.CheckArtifactVersionDeletionForLicenses(ctx, artifact.ID, version, versionsWithSameDigest); err != nil {
			return err
		}

		// Step 4: Check if this is the last non-SHA tag of the artifact
		isLast, err := db.IsLastTagOfArtifact(ctx, artifact.ID, tagName)
		if err != nil {
			return err
		}
		if isLast {
			return apierrors.NewConflict(
				"Cannot delete tag: it is the last tag of the artifact. At least one tag must remain for the artifact.",
			)
		}

		// Step 5: Delete the tag
		return db.DeleteArtifactVersion(ctx, artifact.ID, tagName)
	})
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if errors.Is(err, apierrors.ErrBadRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		log.Error("error deleting artifact tag", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func artifactMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		var artifact *types.ArtifactWithTaggedVersion
		var err error

		if artifactId, parseErr := uuid.Parse(r.PathValue("artifactId")); parseErr != nil {
			http.NotFound(w, r)
			return
		} else if auth.CurrentOrg().HasFeature(types.FeatureLicensing) && auth.CurrentCustomerOrgID() != nil {
			artifact, err = db.GetArtifactByID(ctx, *auth.CurrentOrgID(), artifactId, auth.CurrentCustomerOrgID())
		} else {
			artifact, err = db.GetArtifactByID(ctx, *auth.CurrentOrgID(), artifactId, nil)
		}

		if err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				log.Error("failed to get artifact", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		} else {
			h.ServeHTTP(w, r.WithContext(internalctx.WithArtifact(ctx, artifact)))
		}
	})
}
