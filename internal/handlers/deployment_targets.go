package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentconnect"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	internalctx "github.com/distr-sh/distr/internal/context"
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
	r.WithOptions(option.GroupTags("Agents"))
	r.Use(middleware.RequireOrgAndRole)
	r.Get("/", getDeploymentTargets).
		With(option.Description("List all deployment targets")).
		With(option.Response(http.StatusOK, []types.DeploymentTargetFull{}))
	r.With(middleware.RequireReadWriteOrAdmin).
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
		r.With(middleware.RequireReadWriteOrAdmin).Group(func(r chiopenapi.Router) {
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
			r.With(middleware.RequireReadWriteOrAdmin).
				Put("/", putDeploymentTargetNotesHandler()).
				With(option.Description("Set notes for this deployment target")).
				With(option.Request(struct {
					DeploymentTargetIDRequest
					api.DeploymentTargetNotesRequest
				}{})).
				With(option.Response(http.StatusOK, api.DeploymentTargetNotes{}))
		})
		r.Get("/logs", getDeploymentTargetLogRecordsHandler()).
			With(option.Description("Get logs for this deployment target")).
			With(option.Request(DeploymentTargetTimeseriesRequest{})).
			With(option.Response(http.StatusOK, []api.DeploymentTargetLogRecord{}))
		r.Get("/logs/export", exportDeploymentTargetLogRecordsHandler()).
			With(option.Description("Get logs for this deployment target")).
			With(option.Request(DeploymentTargetIDRequest{})).
			With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
	})
}

func getDeploymentTargets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	deploymentTargets, err := db.GetDeploymentTargets(
		ctx,
		*auth.CurrentOrgID(),
		auth.CurrentCustomerOrgID(),
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
	} else if agentVersion, err := db.GetCurrentAgentVersion(ctx); err != nil {
		log.Warn("could not get current agent version", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		dt.AgentVersionID = &agentVersion.ID
		err = db.RunTx(ctx, func(ctx context.Context) error {
			customerOrgID := auth.CurrentCustomerOrgID()

			if dt.CustomerOrganization != nil && dt.CustomerOrganization.ID != uuid.Nil {
				if err := db.ValidateCustomerOrgBelongsToOrg(ctx, dt.CustomerOrganization.ID, *auth.CurrentOrgID()); err != nil {
					err = errors.New("customer organization does not belong to organization")
					http.Error(w, err.Error(), http.StatusForbidden)
					return err
				}
				customerOrgID = &dt.CustomerOrganization.ID
			}

			limitReached, err := subscription.IsDeploymentTargetLimitReached(
				ctx, *auth.CurrentOrg(),
				customerOrgID)
			if err != nil {
				log.Warn("could not check deployment target limit", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			} else if limitReached {
				err = errors.New("deployment target limit reached")
				http.Error(w, err.Error(), http.StatusForbidden)
				return err
			}

			if err = db.CreateDeploymentTarget(
				ctx,
				&dt,
				*auth.CurrentOrgID(),
				auth.CurrentUserID(),
				customerOrgID,
			); err != nil {
				log.Warn("could not create DeploymentTarget", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}

			return nil
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			RespondJSON(w, dt)
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

	if dt.AgentVersion.ID != uuid.Nil {
		dt.AgentVersionID = &dt.AgentVersion.ID
	}

	existing := internalctx.GetDeploymentTarget(ctx)
	if dt.ID == uuid.Nil {
		dt.ID = existing.ID
	} else if dt.ID != existing.ID {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "wrong id")
		return
	}

	if err := db.UpdateDeploymentTarget(ctx, &dt, *auth.CurrentOrgID()); err != nil {
		log.Warn("could not update DeploymentTarget", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err)
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
	} else if !isDeploymentTargetVisible(auth, dt.DeploymentTarget) {
		http.Error(w, "must be vendor or creator", http.StatusForbidden)
	} else if err := db.DeleteDeploymentTargetWithID(ctx, dt.ID); err != nil {
		log.Warn("error deleting deployment target", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		deploymentTarget.AccessKeySalt = &salt
		deploymentTarget.AccessKeyHash = &hash
	}

	if err := db.UpdateDeploymentTargetAccess(ctx, &deploymentTarget.DeploymentTarget, *auth.CurrentOrgID()); err != nil {
		log.Warn("could not update DeploymentTarget", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	org := auth.CurrentOrg()
	connectUrl, err := agentconnect.BuildConnectURL(deploymentTarget.ID, *org, targetSecret)
	if err != nil {
		log.Error("could not create connecturl", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	connectCommand, err := agentconnect.GenerateConnectCommand(
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
		if deploymentTarget, err := db.GetDeploymentTarget(ctx, id, orgId); errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if !isDeploymentTargetVisible(auth, deploymentTarget.DeploymentTarget) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get DeploymentTarget", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			ctx = internalctx.WithDeploymentTarget(ctx, deploymentTarget)
			wh.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func isDeploymentTargetVisible(auth authinfo.AuthInfo, target types.DeploymentTarget) bool {
	return auth.CurrentCustomerOrgID() == nil || util.PtrEq(auth.CurrentCustomerOrgID(), target.CustomerOrganizationID)
}
