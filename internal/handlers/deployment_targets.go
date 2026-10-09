package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/controllerconnect"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func DeploymentTargetsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Controllers"))
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", getDeploymentTargets).
		With(option.Description("List all deployment targets")).
		With(option.Response(http.StatusOK, []types.DeploymentTargetFull{}))
	r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", createDeploymentTarget).
		With(option.Description("Create a new deployment target")).
		With(option.Response(http.StatusOK, []types.DeploymentTargetFull{}))
	r.Route("/{deploymentTargetId}", func(r chiopenapi.Router) {
		type DeploymentTargetIDRequest struct {
			DeploymentTargetID uuid.UUID `path:"deploymentTargetId"`
		}

		type DeploymentTargetTimeseriesRequest struct {
			DeploymentTargetIDRequest
			TimeseriesRequest
		}

		r.Use(deploymentTargetMiddleware)
		r.Get("/", getDeploymentTarget).
			With(option.Description("Get a deployment target")).
			With(option.Request(DeploymentTargetIDRequest{})).
			With(option.Response(http.StatusOK, []types.DeploymentTargetFull{}))
		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
			r.Put("/", updateDeploymentTarget).
				With(option.Description("Update a deployment target")).
				With(option.Request(struct {
					DeploymentTargetIDRequest
					types.DeploymentTargetFull
				}{})).
				With(option.Response(http.StatusOK, []types.DeploymentTargetFull{}))
			r.Delete("/", deleteDeploymentTarget).
				With(option.Description("Delete a deployment target")).
				With(option.Request(DeploymentTargetIDRequest{}))
			r.Post("/access-request", createAccessForDeploymentTarget).
				With(option.Description("Create access token for deployment target")).
				With(option.Request(DeploymentTargetIDRequest{})).
				With(option.Response(http.StatusOK, api.DeploymentTargetAccessTokenResponse{}))
		})
		r.Route("/notes", func(r chiopenapi.Router) {
			r.Get("/", getDeploymentTargetNotesHandler()).
				With(option.Description("Get notes for this deployment target")).
				With(option.Request(DeploymentTargetIDRequest{})).
				With(option.Response(http.StatusOK, api.DeploymentTargetNotes{}))
			r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Put("/", putDeploymentTargetNotesHandler()).
				With(option.Description("Set notes for this deployment target")).
				With(option.Request(struct {
					DeploymentTargetIDRequest
					api.DeploymentTargetNotesRequest
				}{})).
				With(option.Response(http.StatusOK, api.DeploymentTargetNotes{}))
		})
		// These are read-only, controller-pushed logs that are safe to serve from the read-only db.
		r.With(middleware.UseReadonlyDB).Group(func(r chiopenapi.Router) {
			r.Get("/logs", getDeploymentTargetLogRecordsHandler()).
				With(option.Description("Get logs for this deployment target")).
				With(option.Request(DeploymentTargetTimeseriesRequest{})).
				With(option.Response(http.StatusOK, []api.DeploymentTargetLogRecord{}))
			r.Get("/logs/export", exportDeploymentTargetLogRecordsHandler()).
				With(option.Description("Export logs for this deployment target")).
				With(option.Request(struct {
					DeploymentTargetIDRequest
					TimeseriesRangeRequest
				}{})).
				With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
		})
	})
}

func getDeploymentTargets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	deploymentTargets, err := db.GetDeploymentTargets(
		ctx,
		*auth.CurrentOrgID(),
		auth.CurrentCustomerOrgID(),
		auth.CurrentPartnerOrgID(),
	)
	if err != nil {
		internalctx.GetLogger(ctx).Error("failed to get DeploymentTargets", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		RespondJSON(w, deploymentTargets)
	}
}

func getDeploymentTarget(w http.ResponseWriter, r *http.Request) {
	dt := internalctx.GetDeploymentTarget(r.Context())
	RespondJSON(w, dt)
}

func createDeploymentTarget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	if dt, err := JsonBody[types.DeploymentTargetFull](w, r); err != nil {
		return
	} else if err = dt.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if controllerVersion, err := db.GetCurrentControllerVersion(ctx); err != nil {
		log.Warn("could not get current controller version", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		dt.ControllerVersionID = &controllerVersion.ID
		if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
			if dt.CustomerOrganization == nil || dt.CustomerOrganization.ID == uuid.Nil {
				http.Error(w, "partner users must assign a deployment target to a customer", http.StatusForbidden)
				return
			}
		}

		err = db.RunTx(ctx, func(ctx context.Context) error {
			customerOrgID := auth.CurrentCustomerOrgID()

			if dt.CustomerOrganization != nil && dt.CustomerOrganization.ID != uuid.Nil {
				if err := db.ValidateCustomerOrgBelongsToOrg(ctx, dt.CustomerOrganization.ID, *auth.CurrentOrgID()); err != nil {
					return apierrors.NewForbidden("customer organization does not belong to organization")
				}
				if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
					co, err := db.GetCustomerOrganizationByID(ctx, dt.CustomerOrganization.ID)
					if err != nil || !util.PtrEq(co.PartnerOrganizationID, partnerOrgID) {
						return apierrors.NewForbidden("customer is not assigned to your partner organization")
					}
				}
				customerOrgID = &dt.CustomerOrganization.ID
			}

			if limitReached, err := subscription.IsDeploymentTargetLimitReached(
				ctx, *auth.CurrentOrg(), customerOrgID,
			); err != nil {
				return fmt.Errorf("could not check deployment target limit: %w", err)
			} else if limitReached {
				return apierrors.NewForbidden("deployment target limit reached")
			}

			return db.CreateDeploymentTarget(ctx, &dt, *auth.CurrentOrgID(), auth.CurrentUserID(), customerOrgID)
		})

		if err == nil {
			RespondJSON(w, dt)
		} else if errors.Is(err, apierrors.ErrForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
		} else {
			respondInternalError(w, r, err, "could not create DeploymentTarget")
		}
	}
}

func updateDeploymentTarget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	dt, err := JsonBody[types.DeploymentTargetFull](w, r)
	if err != nil {
		return
	}

	if dt.ControllerVersion.ID != uuid.Nil {
		dt.ControllerVersionID = &dt.ControllerVersion.ID
	} else if dt.AutomaticUpdatesEnabled {
		// Without this, enabling automatic updates would only take effect on the next restart.
		controllerVersion, err := db.GetCurrentControllerVersion(ctx)
		if err != nil {
			log.Warn("could not get current controller version", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		dt.ControllerVersionID = &controllerVersion.ID
	}

	existing := internalctx.GetDeploymentTarget(ctx)
	if dt.ID == uuid.Nil {
		dt.ID = existing.ID
	} else if dt.ID != existing.ID {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "wrong id")
		return
	}

	if err := dt.ValidateUpdatableFields(existing.Type); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := db.UpdateDeploymentTarget(ctx, &dt, *auth.CurrentOrgID()); errors.Is(err, apierrors.ErrBadRequest) {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else if err != nil {
		respondInternalError(w, r, err, "could not update DeploymentTarget")
	} else if err = json.NewEncoder(w).Encode(dt); err != nil {
		log.Error("failed to encode json", zap.Error(err))
	}
}

func deleteDeploymentTarget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	dt := internalctx.GetDeploymentTarget(ctx)
	auth := auth.Authentication.Require(ctx)
	if dt.OrganizationID != *auth.CurrentOrgID() {
		http.NotFound(w, r)
	} else if !isDeploymentTargetVisible(ctx, dt) {
		http.Error(w, "must be vendor or creator", http.StatusForbidden)
	} else if err := db.DeleteDeploymentTargetWithID(ctx, dt.ID); err != nil {
		log.Warn("error deleting deployment target", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func createAccessForDeploymentTarget(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)
	auth := auth.Authentication.Require(ctx)

	var targetSecret string
	var err error
	if targetSecret, err = security.GenerateAccessKey(); err != nil {
		log.Error("failed to generate access key", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if salt, hash, err := security.HashAccessKey(targetSecret); err != nil {
		log.Error("failed to hash access key", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else {
		deploymentTarget.PendingAccessKeySalt = &salt
		deploymentTarget.PendingAccessKeyHash = &hash
	}

	if err := db.UpdateDeploymentTargetPendingAccess(
		ctx, &deploymentTarget.DeploymentTarget, *auth.CurrentOrgID(),
	); err != nil {
		log.Warn("could not update DeploymentTarget", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	org := auth.CurrentOrgWithBranding()
	connectUrl, err := controllerconnect.BuildConnectURL(ctx, deploymentTarget.ID, *org, targetSecret)
	if err != nil {
		log.Error("could not create connecturl", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	connectCommand, err := controllerconnect.GenerateConnectCommand(
		ctx,
		deploymentTarget.DeploymentTarget,
		*org,
		targetSecret,
	)
	if err != nil {
		log.Error("could not create connect command", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err = json.NewEncoder(w).Encode(api.DeploymentTargetAccessTokenResponse{
		ConnectURL:     connectUrl,
		TargetID:       deploymentTarget.ID,
		TargetSecret:   targetSecret,
		ConnectCommand: connectCommand,
	}); err != nil {
		log.Error("failed to encode json", zap.Error(err))
	}
}

func deploymentTargetMiddleware(wh http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id, err := uuid.Parse(r.PathValue("deploymentTargetId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		auth := auth.Authentication.Require(ctx)
		orgId := auth.CurrentOrgID()
		deploymentTarget, err := db.GetDeploymentTarget(ctx, id, orgId, auth.CurrentPartnerOrgID())
		if errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get DeploymentTarget", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		} else if !isDeploymentTargetVisible(ctx, deploymentTarget) {
			http.NotFound(w, r)
		} else {
			ctx = internalctx.WithDeploymentTarget(ctx, deploymentTarget)
			wh.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func isDeploymentTargetVisible(ctx context.Context, target *types.DeploymentTargetFull) bool {
	auth := auth.Authentication.Require(ctx)

	if customerOrgID := auth.CurrentCustomerOrgID(); customerOrgID != nil {
		return util.PtrEq(customerOrgID, target.CustomerOrganizationID)
	}

	if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		if target.CustomerOrganization == nil {
			return false
		}
		return util.PtrEq(partnerOrgID, target.CustomerOrganization.PartnerOrganizationID)
	}

	return true
}
