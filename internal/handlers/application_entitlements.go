package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

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

func ApplicationEntitlementsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Applications", "Licensing"))
	r.Use(middleware.RequireOrgAndRole, middleware.LicensingFeatureFlagEnabledMiddleware)
	r.Get("/", getApplicationEntitlements).
		With(option.Description("List all application entitlements")).
		With(option.Response(http.StatusOK, []types.ApplicationEntitlement{}))
	r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", createApplicationEntitlement).
		With(option.Description("Create a new application entitlement")).
		With(option.Request(types.ApplicationEntitlementWithVersions{}))
	r.With(applicationEntitlementMiddleware).Route("/{applicationEntitlementId}", func(r chiopenapi.Router) {
		type ApplicationEntitlementRequest struct {
			ApplicationEntitlementId string `path:"applicationEntitlementId"`
		}

		r.Get("/", getApplicationEntitlement).
			With(option.Description("Get an application entitlement")).
			With(option.Request(ApplicationEntitlementRequest{})).
			With(option.Response(http.StatusOK, types.ApplicationEntitlement{}))
		r.With(middleware.RequireVendor, middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
			Group(func(r chiopenapi.Router) {
				r.Delete("/", deleteApplicationEntitlement).
					With(option.Description("Delete an application entitlement")).
					With(option.Request(ApplicationEntitlementRequest{}))
				r.Put("/", updateApplicationEntitlement).
					With(option.Description("Update an application entitlement")).
					With(option.Request(struct {
						ApplicationEntitlementRequest
						types.ApplicationEntitlementWithVersions
					}{})).
					With(option.Response(http.StatusOK, types.ApplicationEntitlement{}))
			})
	})
}

func createApplicationEntitlement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	entitlement, err := JsonBody[types.ApplicationEntitlementWithVersions](w, r)
	if err != nil {
		return
	}
	entitlement.OrganizationID = *auth.CurrentOrgID()

	sanitizeRegistryInput(entitlement)

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		err := db.CreateApplicationEntitlement(ctx, &entitlement.ApplicationEntitlementBase)
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "An entitlement with this name already exists", http.StatusBadRequest)
			return err
		} else if err != nil {
			log.Warn("could not create entitlement", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		for _, version := range entitlement.Versions {
			if err := db.AddVersionToApplicationEntitlement(
				ctx, &entitlement.ApplicationEntitlementBase, version.ID,
			); err != nil {
				log.Warn("could not add version to entitlement", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		}

		if createdEntitlement, err := db.GetApplicationEntitlementByID(ctx, entitlement.ID); err != nil {
			log.Warn("could not read previously created entitlement", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			return err
		} else {
			RespondJSON(w, createdEntitlement)
		}

		return nil
	})
}

func updateApplicationEntitlement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	entitlement, err := JsonBody[types.ApplicationEntitlementWithVersions](w, r)
	if err != nil {
		return
	}
	entitlement.OrganizationID = *auth.CurrentOrgID()

	existing := internalctx.GetApplicationEntitlement(ctx)
	if entitlement.ID == uuid.Nil {
		entitlement.ID = existing.ID
	} else if entitlement.ID != existing.ID {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if existing.CustomerOrganizationID != nil &&
		(entitlement.CustomerOrganizationID == nil ||
			*existing.CustomerOrganizationID != *entitlement.CustomerOrganizationID) {
		http.Error(w, "Changing the entitlement owner is not allowed", http.StatusBadRequest)
		return
	} else if existing.ApplicationID != entitlement.ApplicationID {
		http.Error(w, "Changing the application is not allowed", http.StatusBadRequest)
		return
	}
	sanitizeRegistryInput(entitlement)

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		err := db.UpdateApplicationEntitlement(ctx, &entitlement.ApplicationEntitlementBase)
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "An entitlement with this name already exists", http.StatusBadRequest)
			return err
		} else if err != nil {
			log.Warn("could not update entitlement", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
			return err
		}

		newVersionIDs := make([]uuid.UUID, len(entitlement.Versions))
		for i, v := range entitlement.Versions {
			newVersionIDs[i] = v.ID
		}

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
				log.Warn("could not check deployment version usage", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			} else if len(conflicts) > 0 {
				msg := formatVersionConflictError(conflicts)
				http.Error(w, msg, http.StatusConflict)
				return errors.New(msg)
			}
		}

		if err := db.SetApplicationEntitlementVersions(ctx, entitlement.ID, newVersionIDs); err != nil {
			log.Warn("could not update entitlement versions", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		if updatedEntitlement, err := db.GetApplicationEntitlementByID(ctx, entitlement.ID); err != nil {
			log.Warn("could not read previously updated entitlement", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			return err
		} else {
			RespondJSON(w, updatedEntitlement)
		}

		return nil
	})
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

func sanitizeRegistryInput(entitlement types.ApplicationEntitlementWithVersions) {
	if entitlement.RegistryURL == nil || (*entitlement.RegistryURL) == "" {
		entitlement.RegistryURL = nil
		entitlement.RegistryUsername = nil
		entitlement.RegistryPassword = nil
	}
}

func getApplicationEntitlements(w http.ResponseWriter, r *http.Request) {
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
		if entitlements, err := db.GetApplicationEntitlementsWithOrganizationID(
			ctx, *auth.CurrentOrgID(), applicationId); err != nil {
			internalctx.GetLogger(ctx).Error("failed to get entitlements", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			RespondJSON(w, entitlements)
		}
	} else {
		if entitlements, err := db.GetApplicationEntitlementsWithCustomerOrganizationID(
			ctx, *auth.CurrentCustomerOrgID(), *auth.CurrentOrgID(), applicationId,
		); err != nil {
			internalctx.GetLogger(ctx).Error("failed to get entitlements", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			RespondJSON(w, entitlements)
		}
	}
}

func getApplicationEntitlement(w http.ResponseWriter, r *http.Request) {
	entitlement := internalctx.GetApplicationEntitlement(r.Context())
	RespondJSON(w, entitlement)
}

func deleteApplicationEntitlement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	entitlement := internalctx.GetApplicationEntitlement(ctx)
	auth := auth.Authentication.Require(ctx)
	if entitlement.OrganizationID != *auth.CurrentOrgID() {
		http.NotFound(w, r)
	} else if err := db.DeleteApplicationEntitlementWithID(ctx, entitlement.ID); errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "could not delete entitlement because it is still in use", http.StatusBadRequest)
	} else if err != nil {
		log.Warn("error deleting entitlement", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func applicationEntitlementMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)
		if entitlementId, err := uuid.Parse(r.PathValue("applicationEntitlementId")); err != nil {
			http.Error(w, "applicationEntitlementId is not a valid UUID", http.StatusBadRequest)
		} else if entitlement, err := db.GetApplicationEntitlementByID(ctx, entitlementId); errors.Is(
			err, apierrors.ErrNotFound,
		) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get entitlement", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else if !canSeeEntitlement(auth, entitlement) {
			w.WriteHeader(http.StatusForbidden)
		} else {
			ctx = internalctx.WithApplicationEntitlement(ctx, entitlement)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
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
	return true
}
