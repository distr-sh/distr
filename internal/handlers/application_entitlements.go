package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/notification"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func ApplicationEntitlementsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Applications", "Licensing"))
	r.Use(middleware.RequireOrgAndRole, middleware.LicensingFeatureFlagEnabledMiddleware)
	r.Get("/", respond.Handler(getApplicationEntitlements)).
		With(option.Description("List all application entitlements")).
		With(option.Response(http.StatusOK, []types.ApplicationEntitlement{}))
	r.With(middleware.RequireVendorOrPartner, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", respond.Handler(createApplicationEntitlement)).
		With(option.Description("Create a new application entitlement")).
		With(option.Request(types.ApplicationEntitlementWithVersions{}))
	r.With(respond.Middleware(applicationEntitlementMiddleware)).
		Route("/{applicationEntitlementId}", func(r chiopenapi.Router) {
			type ApplicationEntitlementRequest struct {
				ApplicationEntitlementId string `path:"applicationEntitlementId"`
			}

			r.Get("/", respond.Handler(getApplicationEntitlement)).
				With(option.Description("Get an application entitlement")).
				With(option.Request(ApplicationEntitlementRequest{})).
				With(option.Response(http.StatusOK, types.ApplicationEntitlement{}))
			r.With(middleware.RequireVendorOrPartner, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Group(func(r chiopenapi.Router) {
					r.Delete("/", respond.Handler(deleteApplicationEntitlement)).
						With(option.Description("Delete an application entitlement")).
						With(option.Request(ApplicationEntitlementRequest{}))
					r.Put("/", respond.Handler(updateApplicationEntitlement)).
						With(option.Description("Update an application entitlement")).
						With(option.Request(struct {
							ApplicationEntitlementRequest
							types.ApplicationEntitlementWithVersions
						}{})).
						With(option.Response(http.StatusOK, types.ApplicationEntitlement{}))
				})
		})
}

var (
	applicationEntitlementNameTaken = respond.BadRequest("An entitlement with this name already exists")
	invalidEntitlementApplication   = respond.BadRequest("invalid application ID")
	invalidEntitlementVersion       = respond.BadRequest("invalid application version ID")
)

func createApplicationEntitlement(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	entitlement, resp := decodeJSON[types.ApplicationEntitlementWithVersions](r)
	if resp != nil {
		return resp, nil
	}
	entitlement.OrganizationID = *auth.CurrentOrgID()

	var created *types.ApplicationEntitlement
	if entitlement.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else if entitlement.ApplicationID == uuid.Nil {
		return respond.BadRequest("application is required"), nil
	} else if _, resp, err := customerScope(ctx, entitlement.CustomerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else if _, err := db.GetApplication(ctx, entitlement.ApplicationID, entitlement.OrganizationID); errors.Is(
		err, distrerr.ErrNotFound,
	) {
		return invalidEntitlementApplication, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	} else if resp, err := checkEntitlementVersions(ctx, entitlement); resp != nil || err != nil {
		return resp, err
	} else if err := checkRegistryInput(&entitlement); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if err := checkAutoAddVersionRegex(&entitlement); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		if err := db.LockApplicationExclusive(ctx, entitlement.ApplicationID); err != nil {
			return nil, err
		} else if err := db.CreateApplicationEntitlement(
			ctx, &entitlement.ApplicationEntitlementBase,
		); errors.Is(err, distrerr.ErrConflict) {
			return applicationEntitlementNameTaken, nil
		} else if err != nil {
			return nil, fmt.Errorf("could not create entitlement: %w", err)
		} else if err := db.SetApplicationEntitlementVersions(
			ctx, entitlement.ID, versionIDs(entitlement.Versions),
		); err != nil {
			return nil, fmt.Errorf("could not add versions to entitlement: %w", err)
		}

		var err error
		if created, err = db.GetApplicationEntitlementByID(ctx, entitlement.ID); err != nil {
			return nil, fmt.Errorf("could not read previously created entitlement: %w", err)
		}
		return nil, nil
	}); resp != nil || err != nil {
		return resp, err
	} else {
		notifyEntitledVersions(ctx, entitlement.ApplicationEntitlementBase, versionIDs(entitlement.Versions))
		return respond.JSON(created), nil
	}
}

func notifyEntitledVersions(
	ctx context.Context,
	entitlement types.ApplicationEntitlementBase,
	applicationVersionIDs []uuid.UUID,
) {
	notification.Dispatch(ctx, func(ctx context.Context) error {
		return notification.SendApplicationEntitlementVersionsNotifications(
			ctx, entitlement.OrganizationID, entitlement.ApplicationID, applicationVersionIDs,
		)
	})
}

func checkEntitlementVersions(
	ctx context.Context, entitlement types.ApplicationEntitlementWithVersions,
) (respond.Responder, error) {
	if ok, err := db.ApplicationVersionsBelongToApplication(
		ctx, entitlement.ApplicationID, versionIDs(entitlement.Versions),
	); err != nil {
		return nil, fmt.Errorf("failed to check application versions: %w", err)
	} else if !ok {
		return invalidEntitlementVersion, nil
	} else {
		return nil, nil
	}
}

func versionIDs(versions []types.ApplicationVersion) []uuid.UUID {
	ids := make([]uuid.UUID, len(versions))
	for i, version := range versions {
		ids[i] = version.ID
	}
	return ids
}

func updateApplicationEntitlement(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	existing := internalctx.GetApplicationEntitlement(ctx)
	entitlement, resp := decodeJSON[types.ApplicationEntitlementWithVersions](r)
	if resp != nil {
		return resp, nil
	}
	entitlement.OrganizationID = existing.OrganizationID
	if entitlement.ID == uuid.Nil {
		entitlement.ID = existing.ID
	}

	if entitlement.Name == "" {
		return respond.BadRequest("name is required"), nil
	} else if entitlement.ID != existing.ID {
		return respond.BadRequest("entitlement ID does not match the URL"), nil
	} else if existing.CustomerOrganizationID != nil &&
		!util.PtrEq(existing.CustomerOrganizationID, entitlement.CustomerOrganizationID) {
		return respond.BadRequest("Changing the entitlement owner is not allowed"), nil
	} else if existing.ApplicationID != entitlement.ApplicationID {
		return respond.BadRequest("Changing the application is not allowed"), nil
	} else if _, resp, err := customerScope(ctx, entitlement.CustomerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else if resp, err := checkEntitlementVersions(ctx, entitlement); resp != nil || err != nil {
		return resp, err
	} else if err := checkRegistryInput(&entitlement); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if err := checkAutoAddVersionRegex(&entitlement); err != nil {
		return respond.BadRequest(err.Error()), nil
	}

	addedVersionIDs, widened := widenedVersions(versionIDs(existing.Versions), versionIDs(entitlement.Versions))

	var updated *types.ApplicationEntitlement
	if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		if err := db.LockApplicationExclusive(ctx, entitlement.ApplicationID); err != nil {
			return nil, err
		} else if err := db.UpdateApplicationEntitlement(
			ctx, &entitlement.ApplicationEntitlementBase,
		); errors.Is(err, distrerr.ErrConflict) {
			return applicationEntitlementNameTaken, nil
		} else if err != nil {
			return nil, fmt.Errorf("could not update entitlement: %w", err)
		}

		newVersionIDs := versionIDs(entitlement.Versions)

		isNarrowing := false
		if len(entitlement.Versions) > 0 {
			if len(existing.Versions) == 0 {
				isNarrowing = true
			} else {
				for _, ev := range existing.Versions {
					if !slices.ContainsFunc(entitlement.Versions, func(v types.ApplicationVersion) bool {
						return v.ID == ev.ID
					}) {
						isNarrowing = true
						break
					}
				}
			}
		}

		if isNarrowing {
			if conflicts, err := db.GetDeploymentsUsingVersionsNotInList(ctx, entitlement.ID, newVersionIDs); err != nil {
				return nil, fmt.Errorf("could not check deployment version usage: %w", err)
			} else if len(conflicts) > 0 {
				return respond.Conflict(formatVersionConflictError(conflicts)), nil
			}
		}

		if err := db.SetApplicationEntitlementVersions(ctx, entitlement.ID, newVersionIDs); err != nil {
			return nil, fmt.Errorf("could not update entitlement versions: %w", err)
		}

		// Widening the entitlement can make a newer version available to the customer.
		if err := triggerAutomaticApplicationUpdates(
			ctx, auth.CurrentOrg(), entitlement.ApplicationID, new(auth.CurrentUserID()),
		); err != nil {
			return nil, fmt.Errorf("failed to trigger automatic application updates: %w", err)
		}

		var err error
		if updated, err = db.GetApplicationEntitlementByID(ctx, entitlement.ID); err != nil {
			return nil, fmt.Errorf("could not read previously updated entitlement: %w", err)
		}
		return nil, nil
	}); resp != nil || err != nil {
		return resp, err
	} else {
		if widened {
			notifyEntitledVersions(ctx, entitlement.ApplicationEntitlementBase, addedVersionIDs)
		}
		return respond.JSON(updated), nil
	}
}

func widenedVersions(existing, updated []uuid.UUID) ([]uuid.UUID, bool) {
	if len(existing) == 0 {
		return nil, false
	} else if len(updated) == 0 {
		return nil, true
	}
	added := slices.DeleteFunc(slices.Clone(updated), func(id uuid.UUID) bool {
		return slices.Contains(existing, id)
	})
	return added, len(added) > 0
}

func formatVersionConflictError(conflicts []types.DeploymentVersionUsage) string {
	details := make([]string, len(conflicts))
	for i, c := range conflicts {
		details[i] = fmt.Sprintf(
			"deployment target %q uses version %q",
			c.DeploymentTargetName, c.ApplicationVersionName,
		)
	}
	return fmt.Sprintf(
		"cannot narrow entitlement scope: %s",
		strings.Join(details, ", "),
	)
}

// checkRegistryInput enforces that either all or none of the registry fields are set.
func checkRegistryInput(entitlement *types.ApplicationEntitlementWithVersions) error {
	if entitlement.RegistryURL != nil && *entitlement.RegistryURL == "" {
		entitlement.RegistryURL = nil
	}
	if entitlement.RegistryUsername != nil && *entitlement.RegistryUsername == "" {
		entitlement.RegistryUsername = nil
	}
	if entitlement.RegistryPassword != nil && *entitlement.RegistryPassword == "" {
		entitlement.RegistryPassword = nil
	}
	if entitlement.RegistryURL == nil {
		if entitlement.RegistryUsername != nil || entitlement.RegistryPassword != nil {
			return errors.New("registry URL is required when registry credentials are given")
		}
	} else if entitlement.RegistryUsername == nil || entitlement.RegistryPassword == nil {
		return errors.New("registry username and password are required when a registry URL is given")
	}
	return nil
}

const maxAutoAddVersionRegexLength = 200

func checkAutoAddVersionRegex(entitlement *types.ApplicationEntitlementWithVersions) error {
	if entitlement.AutoAddVersionRegex != nil && *entitlement.AutoAddVersionRegex == "" {
		entitlement.AutoAddVersionRegex = nil
	}
	if entitlement.AutoAddVersionRegex == nil {
		return nil
	} else if len(entitlement.Versions) == 0 {
		return errors.New("an auto-add version regex requires at least one selected version")
	} else if _, err := validation.ParseRegex(*entitlement.AutoAddVersionRegex, maxAutoAddVersionRegexLength); err != nil {
		return err
	}
	return nil
}

func getApplicationEntitlements(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	var applicationId *uuid.UUID
	if applicationidParam := r.URL.Query().Get("applicationId"); applicationidParam != "" {
		if id, err := uuid.Parse(applicationidParam); err != nil {
			return respond.BadRequest("applicationId is not a valid UUID"), nil
		} else {
			applicationId = &id
		}
	}
	var (
		entitlements []types.ApplicationEntitlement
		err          error
	)
	if auth.CurrentCustomerOrgID() != nil {
		entitlements, err = db.GetApplicationEntitlementsWithCustomerOrganizationID(
			ctx, *auth.CurrentCustomerOrgID(), *auth.CurrentOrgID(), applicationId,
		)
	} else if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		entitlements, err = db.GetApplicationEntitlementsByPartnerOrgID(
			ctx, *partnerOrgID, *auth.CurrentOrgID(), applicationId,
		)
	} else {
		entitlements, err = db.GetApplicationEntitlementsWithOrganizationID(ctx, *auth.CurrentOrgID(), applicationId)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entitlements: %w", err)
	} else {
		return respond.JSON(entitlements), nil
	}
}

func getApplicationEntitlement(r *http.Request) (respond.Responder, error) {
	entitlement := internalctx.GetApplicationEntitlement(r.Context())
	return respond.JSON(entitlement), nil
}

func deleteApplicationEntitlement(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	entitlement := internalctx.GetApplicationEntitlement(ctx)
	auth := auth.Authentication.Require(ctx)
	if entitlement.OrganizationID != *auth.CurrentOrgID() {
		return applicationEntitlementNotFound, nil
	} else if err := db.DeleteApplicationEntitlementWithID(ctx, entitlement.ID); errors.Is(err, distrerr.ErrConflict) {
		return respond.BadRequest("could not delete entitlement because it is still in use"), nil
	} else if err != nil {
		return nil, fmt.Errorf("error deleting entitlement: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func applicationEntitlementMiddleware(r *http.Request) (*http.Request, respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if entitlementId, err := uuid.Parse(r.PathValue("applicationEntitlementId")); err != nil {
		return nil, respond.BadRequest("applicationEntitlementId is not a valid UUID"), nil
	} else if entitlement, err := db.GetApplicationEntitlementByID(ctx, entitlementId); errors.Is(
		err, distrerr.ErrNotFound,
	) {
		return nil, applicationEntitlementNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get entitlement: %w", err)
	} else if !canSeeEntitlement(auth, entitlement) {
		return nil, applicationEntitlementNotFound, nil
	} else {
		return r.WithContext(internalctx.WithApplicationEntitlement(ctx, entitlement)), nil, nil
	}
}

func canSeeEntitlement(auth authinfo.AuthInfo, entitlement *types.ApplicationEntitlement) bool {
	if entitlement.OrganizationID != *auth.CurrentOrgID() {
		return false
	}
	if auth.CurrentCustomerOrgID() != nil {
		if entitlement.CustomerOrganizationID == nil || *entitlement.CustomerOrganizationID != *auth.CurrentCustomerOrgID() {
			return false
		}
	}
	if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		if entitlement.CustomerOrganization == nil ||
			!util.PtrEq(entitlement.CustomerOrganization.PartnerOrganizationID, partnerOrgID) {
			return false
		}
	}
	return true
}
