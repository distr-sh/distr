package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/registry/upstream"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

type UpstreamTagSyncer interface {
	SyncArtifactTags(ctx context.Context, artifact *types.Artifact, skipExistingTags bool) error
}

func ArtifactsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Artifacts"))
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", respond.Handler(getArtifacts)).
		With(option.Description("List all artifacts")).
		With(option.Response(http.StatusOK, []api.ArtifactsResponse{}))
	r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", respond.Handler(createArtifactHandler)).
		With(option.Description("Create an artifact")).
		With(option.Request(api.CreateArtifactRequest{})).
		With(option.Response(http.StatusCreated, api.ArtifactResponse{}))
	r.With(respond.Middleware(artifactMiddleware)).Route("/{artifactId}", func(r chiopenapi.Router) {
		type ArtifactRequest struct {
			ArtifactID uuid.UUID `path:"artifactId"`
		}

		r.Get("/", respond.Handler(getArtifact)).
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
				r.Patch("/", respond.Handler(patchArtifactHandler)).
					With(option.Description(
						"Update artifact upstream URL and/or authentication, or its public visibility")).
					With(option.Request(struct {
						ArtifactRequest
						api.PatchArtifactRequest
					}{})).
					With(option.Response(http.StatusOK, api.ArtifactResponse{}))
				r.Post("/sync", respond.Handler(syncArtifactHandler)).
					With(option.Description("Trigger upstream sync for a pull-through artifact")).
					With(option.Request(ArtifactRequest{})).
					With(option.Response(http.StatusOK, api.ArtifactResponse{}))
				r.Delete("/", respond.Handler(deleteArtifactHandler)).
					With(option.Description("Delete an artifact")).
					With(option.Request(ArtifactRequest{}))
				r.Delete("/tags/{tagName}", respond.Handler(deleteArtifactTagHandler)).
					With(option.Description("Delete an artifact tag")).
					With(option.Request(struct {
						ArtifactRequest
						TagName string `path:"tagName"`
					}{}))
			})
	})
}

func createArtifactHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authentication := auth.Authentication.Require(ctx)

	if body, resp := decodeJSON[api.CreateArtifactRequest](r); resp != nil {
		return resp, nil
	} else if body.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else {
		if body.UpstreamURL != nil {
			if *body.UpstreamURL == "" {
				return respond.BadRequest("upstreamUrl must not be empty"), nil
			} else if err := upstream.ValidateUpstreamURL(ctx, *body.UpstreamURL); err != nil {
				return respond.BadRequest(err.Error()), nil
			}
		}

		orgID := *authentication.CurrentOrgID()
		if org, err := db.GetOrganizationByID(ctx, orgID); err != nil {
			return nil, fmt.Errorf("failed to get organization: %w", err)
		} else if org.Slug == nil || *org.Slug == "" {
			return respond.BadRequest(
				"your organization does not have a slug yet. Please set one in the organization settings " +
					"before creating an artifact.",
			), nil
		} else {
			artifact := &types.Artifact{
				Name:           body.Name,
				OrganizationID: orgID,
				UpstreamURL:    body.UpstreamURL,
			}
			if body.UpstreamAuth != nil {
				if artifact.UpstreamURL == nil {
					return respond.BadRequest("upstream auth can only be set if upstream URL is set"), nil
				}

				artifact.UpstreamAuthType = &body.UpstreamAuth.Type
				artifact.UpstreamUsername = dbcrypto.StringPtr(body.UpstreamAuth.Username)
				artifact.UpstreamPassword = dbcrypto.StringPtr(body.UpstreamAuth.Password)
			}
			if err := upstream.ValidateUpstreamCredentials(ctx, artifact); err != nil {
				return respond.BadRequest(err.Error()), nil
			} else if err := db.CreateArtifact(ctx, artifact); errors.Is(err, apierrors.ErrConflict) {
				return respond.Conflict("artifact name already exists"), nil
			} else if err != nil {
				return nil, fmt.Errorf("failed to create artifact: %w", err)
			} else if result, err := db.GetArtifactByID(ctx, artifact.OrganizationID, artifact.ID, nil); err != nil {
				return nil, fmt.Errorf("failed to fetch created artifact: %w", err)
			} else {
				return respond.JSON(mapping.ArtifactToAPI(*result)).WithStatus(http.StatusCreated), nil
			}
		}
	}
}

func syncArtifactHandler(r *http.Request) (respond.Responder, error) {
	syncer := new(upstream.Syncer)
	ctx := r.Context()
	artifact := internalctx.GetArtifact(ctx)

	if artifact.UpstreamURL == nil {
		return respond.BadRequest("artifact is not a pull-through cache"), nil
	} else if err := syncer.SyncArtifactTags(ctx, &artifact.Artifact, false); err != nil {
		return nil, fmt.Errorf("upstream sync failed: %w", err)
	} else if result, err := db.GetArtifactByID(ctx, artifact.OrganizationID, artifact.ID, nil); err != nil {
		return nil, fmt.Errorf("failed to fetch artifact after sync: %w", err)
	} else {
		return respond.JSON(mapping.ArtifactToAPI(*result)), nil
	}
}

func getArtifacts(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	var artifacts []types.ArtifactWithDownloads
	var err error
	if auth.CurrentOrg().HasFeature(types.FeatureLicensing) && auth.CurrentCustomerOrgID() != nil {
		if entitlements, err1 := db.GetArtifactEntitlements(ctx, *auth.CurrentOrgID()); err1 != nil {
			return nil, fmt.Errorf("failed to get artifact entitlements: %w", err1)
		} else if len(entitlements) > 0 {
			artifacts, err = db.GetArtifactsByEntitlementOwnerID(ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID())
		} else {
			artifacts, err = db.GetArtifactsByOrgID(ctx, *auth.CurrentOrgID())
		}
	} else {
		artifacts, err = db.GetArtifactsByOrgID(ctx, *auth.CurrentOrgID())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	} else {
		return respond.JSON(mapping.List(artifacts, mapping.ArtifactsWithDownloadsToAPI)), nil
	}
}

func getArtifact(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	return respond.JSON(mapping.ArtifactToAPI(*internalctx.GetArtifact(ctx))), nil
}

var patchImageArtifactHandler = patchImageHandler(func(ctx context.Context, body api.PatchImageRequest) (any, error) {
	artifact := internalctx.GetArtifact(ctx)
	if err := db.UpdateArtifactImage(ctx, artifact, body.ImageID); err != nil {
		return nil, err
	} else {
		return mapping.ArtifactToAPI(*artifact), nil
	}
})

func deleteArtifactHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	artifact := internalctx.GetArtifact(ctx)

	if err := db.RunTx(ctx, func(ctx context.Context) error {
		if isReferenced, err := db.ArtifactIsReferencedInEntitlements(ctx, artifact.ID); err != nil {
			return err
		} else if isReferenced {
			return apierrors.NewBadRequest("Cannot delete artifact: it is referenced in one or more entitlements.")
		} else if err := db.DeleteArtifactWithID(ctx, artifact.ID); err != nil {
			return err
		} else {
			return nil
		}
	}); errors.Is(err, apierrors.ErrNotFound) {
		return artifactNotFound, nil
	} else if errors.Is(err, apierrors.ErrBadRequest) {
		return respond.BadRequest(err.Error()), nil
	} else if err != nil {
		return nil, fmt.Errorf("error deleting artifact: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func deleteArtifactTagHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	artifact := internalctx.GetArtifact(ctx)

	if tagName := r.PathValue("tagName"); tagName == "" {
		return respond.NotFound("artifact tag not found"), nil
	} else if err := db.DeleteArtifactTag(ctx, artifact.ID, tagName); errors.Is(err, apierrors.ErrNotFound) {
		return respond.NotFound("artifact tag not found"), nil
	} else if errors.Is(err, apierrors.ErrBadRequest) {
		return respond.BadRequest(err.Error()), nil
	} else if errors.Is(err, apierrors.ErrConflict) {
		return respond.Conflict(err.Error()), nil
	} else if err != nil {
		return nil, fmt.Errorf("error deleting artifact tag: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func patchArtifactHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	artifact := internalctx.GetArtifact(ctx)

	body, resp := decodeJSON[api.PatchArtifactRequest](r)
	if resp != nil {
		return resp, nil
	}

	if body.Public != nil {
		if role := auth.Authentication.Require(ctx).CurrentUserRole(); role == nil || *role != types.UserRoleAdmin {
			return respond.Forbidden("must be admin to change the visibility of an artifact"), nil
		}
	}

	params := db.UpdateArtifactUpstreamParams{}

	if body.UpstreamURL != nil {
		if *body.UpstreamURL == "" {
			return respond.BadRequest("upstreamUrl must not be empty"), nil
		} else if err := upstream.ValidateUpstreamURL(ctx, *body.UpstreamURL); err != nil {
			return respond.BadRequest(err.Error()), nil
		} else {
			params.UpdateURL = true
			params.UpstreamURL = body.UpstreamURL
		}
	}

	if body.Auth.Present {
		params.UpdateAuth = true
		if body.Auth.Value != nil {
			params.AuthType = &body.Auth.Value.Type
			params.Username = dbcrypto.StringPtr(body.Auth.Value.Username)
			params.Password = dbcrypto.StringPtr(body.Auth.Value.Password)
		}
	}

	if params.UpdateURL || params.UpdateAuth {
		if artifact.UpstreamURL == nil {
			return respond.BadRequest("artifact is not a pull-through cache artifact"), nil
		}
		validationArtifact := artifact.Artifact
		if params.UpdateURL {
			validationArtifact.UpstreamURL = params.UpstreamURL
		}
		if params.UpdateAuth {
			validationArtifact.UpstreamAuthType = params.AuthType
			validationArtifact.UpstreamUsername = params.Username
			validationArtifact.UpstreamPassword = params.Password
		}
		if err := upstream.ValidateUpstreamCredentials(ctx, &validationArtifact); err != nil {
			return respond.BadRequest(err.Error()), nil
		}
	}

	if err := db.RunTx(ctx, func(ctx context.Context) error {
		if params.UpdateURL || params.UpdateAuth {
			if err := db.UpdateArtifactUpstream(ctx, artifact.ID, artifact.OrganizationID, params); err != nil {
				return err
			}
		}
		if body.Public != nil {
			return db.UpdateArtifactPublic(ctx, artifact.ID, artifact.OrganizationID, *body.Public)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to update artifact: %w", err)
	} else if result, err := db.GetArtifactByID(ctx, artifact.OrganizationID, artifact.ID, nil); err != nil {
		return nil, fmt.Errorf("failed to fetch artifact after update: %w", err)
	} else {
		return respond.JSON(mapping.ArtifactToAPI(*result)), nil
	}
}

var artifactNotFound = respond.NotFound("artifact not found")

func artifactMiddleware(r *http.Request) (*http.Request, respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	var entitledCustomerOrgID *uuid.UUID
	if auth.CurrentOrg().HasFeature(types.FeatureLicensing) {
		entitledCustomerOrgID = auth.CurrentCustomerOrgID()
	}

	if artifactId, err := uuid.Parse(r.PathValue("artifactId")); err != nil {
		return nil, artifactNotFound, nil
	} else if artifact, err := db.GetArtifactByID(
		ctx, *auth.CurrentOrgID(), artifactId, entitledCustomerOrgID,
	); errors.Is(err, apierrors.ErrNotFound) {
		return nil, artifactNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get artifact: %w", err)
	} else {
		return r.WithContext(internalctx.WithArtifact(ctx, artifact)), nil, nil
	}
}
