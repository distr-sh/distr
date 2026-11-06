package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/notification"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func ArtifactEntitlementsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Artifacts", "Licensing"))
	r.Use(middleware.RequireOrgAndRole, middleware.LicensingFeatureFlagEnabledMiddleware)
	r.Get("/", respond.Handler(getArtifactEntitlements)).
		With(option.Description("List all artifact entitlements")).
		With(option.Response(http.StatusOK, []types.ArtifactEntitlement{}))
	r.With(
		middleware.RequireVendorOrPartner, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin,
	).Group(func(r chiopenapi.Router) {
		r.Post("/", respond.Handler(createArtifactEntitlement)).
			With(option.Description("Create a new artifact entitlement")).
			With(option.Request(types.ArtifactEntitlement{})).
			With(option.Response(http.StatusOK, types.ArtifactEntitlement{}))
		r.With(respond.Middleware(artifactEntitlementMiddleware)).
			Route("/{artifactEntitlementId}", func(r chiopenapi.Router) {
				type ArtifactEntitlementRequest struct {
					ArtifactEntitlementID uuid.UUID `path:"artifactEntitlementId"`
				}

				r.Put("/", respond.Handler(updateArtifactEntitlement)).
					With(option.Description("Update an artifact entitlement")).
					With(option.Request(struct {
						ArtifactEntitlementRequest
						types.ArtifactEntitlement
					}{})).
					With(option.Response(http.StatusOK, types.ArtifactEntitlement{}))
				r.Delete("/", respond.Handler(deleteArtifactEntitlement)).
					With(option.Description("Delete an artifact entitlement")).
					With(option.Request(ArtifactEntitlementRequest{}))
			})
	})
}

func getArtifactEntitlements(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)

	var (
		entitlements []types.ArtifactEntitlement
		err          error
	)
	if customerOrgID := authCtx.CurrentCustomerOrgID(); customerOrgID != nil {
		entitlements, err = db.GetArtifactEntitlementsByCustomerOrgID(ctx, *customerOrgID, *authCtx.CurrentOrgID())
	} else if partnerOrgID := authCtx.CurrentPartnerOrgID(); partnerOrgID != nil {
		entitlements, err = db.GetArtifactEntitlementsByPartnerOrgID(ctx, *partnerOrgID, *authCtx.CurrentOrgID())
	} else {
		entitlements, err = db.GetArtifactEntitlements(ctx, *authCtx.CurrentOrgID())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact entitlements: %w", err)
	} else {
		return respond.JSON(entitlements), nil
	}
}

var artifactEntitlementNameTaken = respond.BadRequest("An artifact entitlement with this name already exists")

func createArtifactEntitlement(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	entitlement, resp := decodeJSON[types.ArtifactEntitlement](r)
	if resp != nil {
		return resp, nil
	}
	entitlement.OrganizationID = *auth.CurrentOrgID()

	if entitlement.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else if _, resp, err := customerScope(ctx, entitlement.CustomerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else if err := validateEntitlementSelections(&entitlement); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		if err := db.LockArtifactRows(ctx, selectionArtifactIDs(entitlement.Artifacts)); err != nil {
			return nil, err
		} else if err := db.CreateArtifactEntitlement(ctx, &entitlement.ArtifactEntitlementBase); errors.Is(
			err, distrerr.ErrConflict,
		) {
			return artifactEntitlementNameTaken, nil
		} else if errors.Is(err, distrerr.ErrBadRequest) {
			return respond.BadRequest(err.Error()), nil
		} else if err != nil {
			return nil, fmt.Errorf("could not create artifact entitlement: %w", err)
		} else {
			return addArtifacts(ctx, entitlement)
		}
	}); resp != nil || err != nil {
		return resp, err
	} else {
		notifyEntitledArtifactVersions(ctx, widenedArtifactEntitlement(nil, entitlement.Artifacts))
		return respond.JSON(entitlement), nil
	}
}

func updateArtifactEntitlement(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	existing := internalctx.GetArtifactEntitlement(ctx)

	entitlement, resp := decodeJSON[types.ArtifactEntitlement](r)
	if resp != nil {
		return resp, nil
	}
	entitlement.OrganizationID = *auth.CurrentOrgID()
	if entitlement.ID == uuid.Nil {
		entitlement.ID = existing.ID
	}

	if entitlement.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else if _, resp, err := customerScope(ctx, entitlement.CustomerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else if err := validateEntitlementSelections(&entitlement); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if entitlement.ID != existing.ID {
		return respond.BadRequest("entitlement ID does not match the URL"), nil
	}

	var widened []types.ArtifactEntitlementSelection
	if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		// No other edit can change the artifacts of a locked entitlement, so unlike those the middleware read, the ones
		// read below are the ones a concurrent registry push may be adding tags to.
		if err := db.LockArtifactEntitlementExclusive(ctx, entitlement.ID); err != nil {
			return nil, err
		}
		current, err := db.GetArtifactEntitlementByID(ctx, entitlement.ID)
		if errors.Is(err, distrerr.ErrNotFound) {
			return artifactEntitlementNotFound, nil
		} else if err != nil {
			return nil, fmt.Errorf("could not read entitlement: %w", err)
		} else if err := db.LockArtifactRows(
			ctx, append(selectionArtifactIDs(current.Artifacts), selectionArtifactIDs(entitlement.Artifacts)...),
		); err != nil {
			return nil, err
		} else if err := db.UpdateArtifactEntitlement(ctx, &entitlement.ArtifactEntitlementBase); errors.Is(
			err, distrerr.ErrNotFound,
		) {
			return artifactEntitlementNotFound, nil
		} else if errors.Is(err, distrerr.ErrConflict) {
			return artifactEntitlementNameTaken, nil
		} else if errors.Is(err, distrerr.ErrBadRequest) {
			return respond.BadRequest(err.Error()), nil
		} else if err != nil {
			return nil, fmt.Errorf("could not update artifact entitlement: %w", err)
		} else if err := db.RemoveAllArtifactsFromEntitlement(ctx, entitlement.ID); err != nil {
			return nil, fmt.Errorf("could not update artifact entitlement selection: %w", err)
		} else if resp, err := addArtifacts(ctx, entitlement); resp != nil || err != nil {
			return resp, err
		} else {
			widened = widenedArtifactEntitlement(current.Artifacts, entitlement.Artifacts)
			return nil, nil
		}
	}); resp != nil || err != nil {
		return resp, err
	} else {
		notifyEntitledArtifactVersions(ctx, widened)
		return respond.JSON(entitlement), nil
	}
}

func selectionArtifactIDs(selections []types.ArtifactEntitlementSelection) []uuid.UUID {
	ids := make([]uuid.UUID, len(selections))
	for i, selection := range selections {
		ids[i] = selection.ArtifactID
	}
	return ids
}

func notifyEntitledArtifactVersions(ctx context.Context, selections []types.ArtifactEntitlementSelection) {
	if len(selections) == 0 {
		return
	}
	notification.Dispatch(ctx, func(ctx context.Context) error {
		var aggErr error
		for _, selection := range selections {
			if err := notification.SendArtifactEntitlementVersionsNotifications(
				ctx, selection.ArtifactID, selection.VersionIDs,
			); err != nil {
				aggErr = errors.Join(aggErr, fmt.Errorf("artifact %v: %w", selection.ArtifactID, err))
			}
		}
		return aggErr
	})
}

func widenedArtifactEntitlement(
	existing, updated []types.ArtifactEntitlementSelection,
) []types.ArtifactEntitlementSelection {
	var widened []types.ArtifactEntitlementSelection
	for _, selection := range updated {
		index := slices.IndexFunc(existing, func(previous types.ArtifactEntitlementSelection) bool {
			return previous.ArtifactID == selection.ArtifactID
		})
		if index < 0 {
			widened = append(widened, selection)
		} else if added, ok := widenedVersions(existing[index].VersionIDs, selection.VersionIDs); ok {
			widened = append(widened, types.ArtifactEntitlementSelection{
				ArtifactID: selection.ArtifactID,
				VersionIDs: added,
			})
		}
	}
	return widened
}

func validateEntitlementSelections(entitlement *types.ArtifactEntitlement) error {
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
	return checkArtifactAutoAddVersionRegex(entitlement)
}

func checkArtifactAutoAddVersionRegex(entitlement *types.ArtifactEntitlement) error {
	if regex := entitlement.AutoAddVersionRegex; regex == nil || *regex == "" {
		entitlement.AutoAddVersionRegex = nil
		return nil
	} else if len(entitlement.Artifacts) != 1 {
		return errors.New("an auto-add tag regex requires an entitlement with exactly one artifact")
	} else if len(entitlement.Artifacts[0].VersionIDs) == 0 {
		return errors.New("an auto-add tag regex requires at least one selected tag")
	}
	return validation.ValidateAutoAddVersionRegex(*entitlement.AutoAddVersionRegex)
}

func addArtifacts(ctx context.Context, entitlement types.ArtifactEntitlement) (respond.Responder, error) {
	add := func(artifactID uuid.UUID, versionID *uuid.UUID) (respond.Responder, error) {
		if err := db.AddArtifactToArtifactEntitlement(
			ctx, entitlement.ID, artifactID, versionID,
		); errors.Is(err, distrerr.ErrBadRequest) {
			return respond.BadRequest(err.Error()), nil
		} else if err != nil {
			return nil, fmt.Errorf("could not add version to entitlement: %w", err)
		} else {
			return nil, nil
		}
	}

	for _, selection := range entitlement.Artifacts {
		if len(selection.VersionIDs) == 0 {
			if resp, err := add(selection.ArtifactID, nil); resp != nil || err != nil {
				return resp, err
			}
		}
		for _, versionID := range selection.VersionIDs {
			if resp, err := add(selection.ArtifactID, &versionID); resp != nil || err != nil {
				return resp, err
			}
		}
	}
	return nil, nil
}

func deleteArtifactEntitlement(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	entitlement := internalctx.GetArtifactEntitlement(ctx)
	auth := auth.Authentication.Require(ctx)
	if entitlement.OrganizationID != *auth.CurrentOrgID() {
		return artifactEntitlementNotFound, nil
	} else if err := db.DeleteArtifactEntitlementWithID(ctx, entitlement.ID); errors.Is(err, distrerr.ErrConflict) {
		return respond.BadRequest("could not delete entitlement because it is still in use"), nil
	} else if err != nil {
		return nil, fmt.Errorf("error deleting entitlement: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func artifactEntitlementMiddleware(r *http.Request) (*http.Request, respond.Responder, error) {
	ctx := r.Context()
	authCtx := auth.Authentication.Require(ctx)
	if entitlementId, err := uuid.Parse(r.PathValue("artifactEntitlementId")); err != nil {
		return nil, respond.BadRequest("artifactEntitlementId is not a valid UUID"), nil
	} else if entitlement, err := db.GetArtifactEntitlementByID(ctx, entitlementId); errors.Is(
		err, distrerr.ErrNotFound,
	) {
		return nil, artifactEntitlementNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get entitlement: %w", err)
	} else if entitlement.OrganizationID != *authCtx.CurrentOrgID() {
		return nil, artifactEntitlementNotFound, nil
	} else if resp, err := checkReachableByPartner(
		ctx, entitlement.CustomerOrganizationID, artifactEntitlementNotFound,
	); resp != nil || err != nil {
		return nil, resp, err
	} else {
		return r.WithContext(internalctx.WithArtifactEntitlement(ctx, entitlement)), nil, nil
	}
}
