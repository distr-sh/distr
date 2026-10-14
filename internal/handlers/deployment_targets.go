package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/controllerconnect"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func DeploymentTargetsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Controllers"))
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", respond.Handler(getDeploymentTargets)).
		With(option.Description("List all deployment targets")).
		With(option.Response(http.StatusOK, []types.DeploymentTargetFull{}))
	r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Post("/", respond.Handler(createDeploymentTarget)).
		With(option.Description("Create a new deployment target")).
		With(option.Request(api.CreateDeploymentTargetRequest{})).
		With(option.Response(http.StatusOK, types.DeploymentTargetFull{}))
	r.Route("/{deploymentTargetId}", func(r chiopenapi.Router) {
		type DeploymentTargetIDRequest struct {
			DeploymentTargetID uuid.UUID `path:"deploymentTargetId"`
		}

		type DeploymentTargetTimeseriesRequest struct {
			DeploymentTargetIDRequest
			TimeseriesRequest
		}

		r.Use(respond.Middleware(deploymentTargetMiddleware))
		r.Get("/", respond.Handler(getDeploymentTarget)).
			With(option.Description("Get a deployment target")).
			With(option.Request(DeploymentTargetIDRequest{})).
			With(option.Response(http.StatusOK, types.DeploymentTargetFull{}))
		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
			r.Put("/", respond.Handler(updateDeploymentTarget)).
				With(option.Description("Update a deployment target")).
				With(option.Request(struct {
					DeploymentTargetIDRequest
					api.UpdateDeploymentTargetRequest
				}{})).
				With(option.Response(http.StatusOK, types.DeploymentTargetFull{}))
			r.Delete("/", respond.Handler(deleteDeploymentTarget)).
				With(option.Description("Delete a deployment target")).
				With(option.Request(DeploymentTargetIDRequest{}))
			r.Post("/access-request", respond.Handler(createAccessForDeploymentTarget)).
				With(option.Description("Create access token for deployment target")).
				With(option.Request(DeploymentTargetIDRequest{})).
				With(option.Response(http.StatusOK, api.DeploymentTargetAccessTokenResponse{}))
			r.Delete("/access-request", respond.Handler(deleteAccessRequestForDeploymentTarget)).
				With(option.Description(
					"Cancel a pending reconnect of a deployment target. " +
						"The pending secret is deleted and the active one stays valid.",
				)).
				With(option.Request(DeploymentTargetIDRequest{}))
		})
		r.Route("/notes", func(r chiopenapi.Router) {
			r.Get("/", respond.Handler(getDeploymentTargetNotesHandler)).
				With(option.Description("Get notes for this deployment target")).
				With(option.Request(DeploymentTargetIDRequest{})).
				With(option.Response(http.StatusOK, api.DeploymentTargetNotes{}))
			r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
				Put("/", respond.Handler(putDeploymentTargetNotesHandler)).
				With(option.Description("Set notes for this deployment target")).
				With(option.Request(struct {
					DeploymentTargetIDRequest
					api.DeploymentTargetNotesRequest
				}{})).
				With(option.Response(http.StatusOK, api.DeploymentTargetNotes{}))
		})
		// These are read-only, controller-pushed logs that are safe to serve from the read-only db.
		r.With(middleware.UseReadonlyDB).Group(func(r chiopenapi.Router) {
			r.Get("/logs", respond.Handler(getDeploymentTargetLogRecordsHandler)).
				With(option.Description("Get logs for this deployment target")).
				With(option.Request(DeploymentTargetTimeseriesRequest{})).
				With(option.Response(http.StatusOK, []api.DeploymentTargetLogRecord{}))
			r.Get("/logs/export", respond.Handler(exportDeploymentTargetLogRecordsHandler)).
				With(option.Description("Export logs for this deployment target")).
				With(option.Request(struct {
					DeploymentTargetIDRequest
					TimeseriesRangeRequest
				}{})).
				With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
		})
	})
}

func getDeploymentTargets(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if deploymentTargets, err := db.GetDeploymentTargets(
		ctx,
		*auth.CurrentOrgID(),
		auth.CurrentCustomerOrgID(),
		auth.CurrentPartnerOrgID(),
	); err != nil {
		return nil, fmt.Errorf("failed to get DeploymentTargets: %w", err)
	} else {
		return respond.JSON(deploymentTargets), nil
	}
}

func getDeploymentTarget(r *http.Request) (respond.Responder, error) {
	dt := internalctx.GetDeploymentTarget(r.Context())
	return respond.JSON(dt), nil
}

func createDeploymentTarget(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if request, resp := decodeJSON[api.CreateDeploymentTargetRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if controllerVersion, err := db.GetCurrentControllerVersion(ctx); err != nil {
		return nil, fmt.Errorf("could not get current controller version: %w", err)
	} else {
		dt := mapping.CreateDeploymentTargetToInternal(request)
		dt.ControllerVersionID = &controllerVersion.ID
		requestedCustomerOrgID := dt.CustomerOrganizationID
		if auth.CurrentPartnerOrgID() != nil && requestedCustomerOrgID == nil {
			return respond.Forbidden("partner users must assign a deployment target to a customer"), nil
		}

		if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
			customerOrgID := auth.CurrentCustomerOrgID()

			if requestedCustomerOrgID != nil {
				if err := db.ValidateCustomerOrgBelongsToOrg(ctx, *requestedCustomerOrgID, *auth.CurrentOrgID()); err != nil {
					return respond.Forbidden("customer organization does not belong to organization"), nil
				}
				if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
					co, err := db.GetCustomerOrganizationByID(ctx, *requestedCustomerOrgID)
					if err != nil || !util.PtrEq(co.PartnerOrganizationID, partnerOrgID) {
						return notAssignedToPartner, nil
					}
				}
				customerOrgID = requestedCustomerOrgID
			}

			if limitReached, err := subscription.IsDeploymentTargetLimitReached(
				ctx, *auth.CurrentOrg(), customerOrgID,
			); err != nil {
				return nil, fmt.Errorf("could not check deployment target limit: %w", err)
			} else if limitReached {
				return respond.Forbidden("deployment target limit reached"), nil
			} else if err := db.CreateDeploymentTarget(
				ctx, &dt, *auth.CurrentOrgID(), auth.CurrentUserID(), customerOrgID,
			); err != nil {
				return nil, fmt.Errorf("could not create DeploymentTarget: %w", err)
			} else {
				return nil, nil
			}
		}); resp != nil || err != nil {
			return resp, err
		} else {
			return respond.JSON(dt), nil
		}
	}
}

func updateDeploymentTarget(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	existing := internalctx.GetDeploymentTarget(ctx)

	if request, resp := decodeJSON[api.UpdateDeploymentTargetRequest](r); resp != nil {
		return resp, nil
	} else if err := request.Validate(existing.Type); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		dt := mapping.UpdateDeploymentTargetToInternal(existing.ID, request)
		if dt.ControllerVersionID == nil && dt.AutomaticUpdatesEnabled {
			// Without this, enabling automatic updates would only take effect on the next restart.
			if controllerVersion, err := db.GetCurrentControllerVersion(ctx); err != nil {
				return nil, fmt.Errorf("could not get current controller version: %w", err)
			} else {
				dt.ControllerVersionID = &controllerVersion.ID
			}
		}

		if err := db.UpdateDeploymentTarget(ctx, &dt, *auth.CurrentOrgID()); errors.Is(err, distrerr.ErrBadRequest) {
			return respond.BadRequest(err.Error()), nil
		} else if err != nil {
			return nil, fmt.Errorf("could not update DeploymentTarget: %w", err)
		} else {
			return respond.JSON(dt), nil
		}
	}
}

func deleteDeploymentTarget(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	dt := internalctx.GetDeploymentTarget(ctx)
	auth := auth.Authentication.Require(ctx)
	if dt.OrganizationID != *auth.CurrentOrgID() {
		return deploymentTargetNotFound, nil
	} else if !isDeploymentTargetVisible(ctx, dt) {
		return respond.Forbidden("must be vendor or creator"), nil
	} else if err := db.DeleteDeploymentTargetWithID(ctx, dt.ID); err != nil {
		return nil, fmt.Errorf("error deleting deployment target: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

func createAccessForDeploymentTarget(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)
	auth := auth.Authentication.Require(ctx)
	org := auth.CurrentOrgWithBranding()

	if targetSecret, err := security.GenerateAccessKey(); err != nil {
		return nil, fmt.Errorf("failed to generate access key: %w", err)
	} else if salt, hash, err := security.HashAccessKey(targetSecret); err != nil {
		return nil, fmt.Errorf("failed to hash access key: %w", err)
	} else {
		deploymentTarget.PendingAccessKeySalt = &salt
		deploymentTarget.PendingAccessKeyHash = &hash

		if err := db.UpdateDeploymentTargetPendingAccess(
			ctx, &deploymentTarget.DeploymentTarget, *auth.CurrentOrgID(),
		); err != nil {
			return nil, fmt.Errorf("could not update DeploymentTarget: %w", err)
		} else if connectUrl, err := controllerconnect.BuildConnectURL(
			ctx, deploymentTarget.ID, *org, targetSecret,
		); err != nil {
			return nil, fmt.Errorf("could not create connecturl: %w", err)
		} else if connectCommand, err := controllerconnect.GenerateConnectCommand(
			ctx,
			deploymentTarget.DeploymentTarget,
			*org,
			targetSecret,
		); err != nil {
			return nil, fmt.Errorf("could not create connect command: %w", err)
		} else {
			return respond.JSON(api.DeploymentTargetAccessTokenResponse{
				ConnectURL:     connectUrl,
				TargetID:       deploymentTarget.ID,
				TargetSecret:   targetSecret,
				ConnectCommand: connectCommand,
			}), nil
		}
	}
}

func deleteAccessRequestForDeploymentTarget(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)
	auth := auth.Authentication.Require(ctx)

	if err := db.DeleteDeploymentTargetPendingAccess(
		ctx, deploymentTarget.ID, *auth.CurrentOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return respond.Conflict("This deployment target has no pending reconnect."), nil
	} else if err != nil {
		return nil, fmt.Errorf("could not delete pending access of DeploymentTarget: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

var deploymentTargetNotFound = respond.NotFound("deployment target not found")

func deploymentTargetMiddleware(r *http.Request) (*http.Request, respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if id, err := uuid.Parse(r.PathValue("deploymentTargetId")); err != nil {
		return nil, deploymentTargetNotFound, nil
	} else if deploymentTarget, err := db.GetDeploymentTarget(
		ctx, id, auth.CurrentOrgID(), auth.CurrentPartnerOrgID(),
	); errors.Is(err, distrerr.ErrNotFound) {
		return nil, deploymentTargetNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get deployment target: %w", err)
	} else if !isDeploymentTargetVisible(ctx, deploymentTarget) {
		return nil, deploymentTargetNotFound, nil
	} else {
		return r.WithContext(internalctx.WithDeploymentTarget(ctx, deploymentTarget)), nil, nil
	}
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
