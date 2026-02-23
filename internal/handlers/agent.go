package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentclient/useragent"
	"github.com/distr-sh/distr/internal/agentconnect"
	"github.com/distr-sh/distr/internal/agentmanifest"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/notification"
	"github.com/distr-sh/distr/internal/security"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func AgentRouter(r chiopenapi.Router) {
	r.With(queryAuthDeploymentTargetCtxMiddleware).Group(func(r chiopenapi.Router) {
		r.WithOptions(option.GroupTags("Agents"))

		type AgentConnectRequest struct {
			TargetID     uuid.UUID `query:"targetId"`
			TargetSecret string    `query:"targetSecret"`
		}

		r.Get("/pre-connect", preConnectHandler()).
			With(option.Request(AgentConnectRequest{})).
			With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
		r.Get("/connect", connectHandler()).
			With(option.Request(AgentConnectRequest{})).
			With(option.Response(http.StatusOK, map[string]any{}, option.ContentType("application/yaml")))
	})

	r.Route("/agent", func(r chiopenapi.Router) {
		r.WithOptions(option.GroupHidden(true))
		// agent login (from basic auth to token)
		r.Post("/login", agentLoginHandler)

		r.With(
			auth.AgentAuthentication.Middleware,
			middleware.AgentSentryUser,
			agentAuthDeploymentTargetCtxMiddleware,
			rateLimitPerAgent,
		).Group(func(r chiopenapi.Router) {
			// agent routes, authenticated via token
			r.Get("/manifest", agentManifestHandler())
			r.Get("/resources", agentResourcesHandler)
			r.Post("/status", agentPostStatusHandler)
			r.Post("/metrics", agentPostMetricsHander)
			r.Put("/logs", agentPutDeploymentLogsHandler())
			r.Put("/deployment-target-logs", agentPutDeploymentTargetLogsHandler())
		})
	})
}

func connectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)

		org, err := db.GetOrganizationByID(ctx, deploymentTarget.OrganizationID)
		if err != nil {
			log.Error("could not get organization for deployment target", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		secret := r.URL.Query().Get("targetSecret")
		if manifest, err := agentmanifest.Get(ctx, *deploymentTarget, *org, &secret); err != nil {
			log.Error("could not get agent manifest", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.Header().Add("Content-Type", "application/yaml")
			if _, err := io.Copy(w, manifest); err != nil {
				log.Warn("writing to client failed", zap.Error(err))
			}
		}
	}
}

// optionally wraps the connect request in a shell script
func preConnectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)

		org, err := db.GetOrganizationByID(ctx, deploymentTarget.OrganizationID)
		if err != nil {
			log.Error("could not get organization for deployment target", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		secret := r.URL.Query().Get("targetSecret")
		script, err := agentconnect.GenerateConnectScript(deploymentTarget.ID, *org, secret)
		if err != nil {
			log.Error("could not generate connect script", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if _, err := w.Write([]byte(script)); err != nil {
			log.Warn("writing to client failed", zap.Error(err))
		}
	}
}

func agentLoginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	if targetID, targetSecret, ok := r.BasicAuth(); !ok {
		log.Info("invalid Basic Auth")
		w.WriteHeader(http.StatusUnauthorized)
	} else if parsedTargetID, err := uuid.Parse(targetID); err != nil {
		http.Error(w, "targetId is not a valid UUID", http.StatusBadRequest)
	} else if agentLoginPerTargetIdRateLimiter.RespondOnLimit(w, r, targetID) {
		return
	} else if deploymentTarget, err := getVerifiedDeploymentTarget(ctx, parsedTargetID, targetSecret); err != nil {
		if errors.Is(err, apierrors.ErrUnauthorized) {
			log.Info("agent unauthorized", zap.Error(err), zap.String("deploymentTargetId", targetID))
			http.Error(w, err.Error(), http.StatusUnauthorized)
		} else {
			log.Error("failed to get deployment target from basic auth", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			sentry.GetHubFromContext(ctx).CaptureException(err)
		}
	} else {
		// TODO maybe even randomize token valid duration
		if _, token, err := authjwt.GenerateAgentTokenValidFor(
			deploymentTarget.ID, deploymentTarget.OrganizationID, env.AgentTokenMaxValidDuration()); err != nil {
			log.Error("failed to create agent token", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if err := json.NewEncoder(w).Encode(api.AuthLoginResponse{Token: token}); err != nil {
				log.Error("failed to write response", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}

func agentResourcesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)
	log := internalctx.GetLogger(ctx).With(zap.String("deploymentTargetId", deploymentTarget.ID.String()))

	statusMessage := "OK"
	deployments, err := db.GetDeploymentsForDeploymentTarget(ctx, deploymentTarget.ID)
	if err != nil {
		msg := "failed to get latest Deployment from DB"
		log.Error(msg, zap.Error(err))
		statusMessage = fmt.Sprintf("%v: %v", msg, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		agentResource := api.AgentResource{
			Version:        deploymentTarget.AgentVersion,
			MetricsEnabled: deploymentTarget.MetricsEnabled,
		}
		if deploymentTarget.Namespace != nil {
			agentResource.Namespace = *deploymentTarget.Namespace
		}

		for _, deployment := range deployments {
			appVersion, err := db.GetApplicationVersion(ctx, deployment.ApplicationVersionID)
			if err != nil {
				msg := "failed to get ApplicationVersion from DB"
				log.Error(msg, zap.Error(err))
				statusMessage = fmt.Sprintf("%v: %v", msg, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				break
			}

			agentDeployment := api.AgentDeployment{
				ID:                 deployment.ID,
				RevisionID:         deployment.DeploymentRevisionID,
				LogsEnabled:        deployment.LogsEnabled,
				ForceRestart:       deployment.ForceRestart,
				IgnoreRevisionSkew: deployment.IgnoreRevisionSkew,
			}

			if deployment.ApplicationLicenseID != nil {
				if license, err := db.GetApplicationLicenseByID(ctx, *deployment.ApplicationLicenseID); err != nil {
					msg := "failed to get ApplicationLicense from DB"
					log.Error(msg, zap.Error(err))
					statusMessage = fmt.Sprintf("%v: %v", msg, err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					break
				} else if license.RegistryURL != nil {
					agentDeployment.RegistryAuth = map[string]api.AgentRegistryAuth{
						*license.RegistryURL: {
							Username: *license.RegistryUsername,
							Password: *license.RegistryPassword,
						},
					}
				}
			}

			var secrets []types.SecretWithUpdatedBy
			if secrets, err = db.GetSecretsForDeploymentTarget(ctx, deploymentTarget.DeploymentTarget); err != nil {
				msg := "failed to get secrets from DB"
				log.Error(msg, zap.Error(err))
				statusMessage = fmt.Sprintf("%v: %v", msg, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				break
			}

			if deploymentTarget.Type == types.DeploymentTypeDocker {
				if composeYaml, err := appVersion.ParsedComposeFile(); err != nil {
					log.Warn("parse error", zap.Error(err))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if patchedComposeFile, err := patchProjectName(composeYaml, deployment.ID); err != nil {
					log.Warn("failed to patch project name", zap.Error(err))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if envFile, err := deploymentvalues.EnvFileReplaceSecrets(&deployment, secrets); err != nil {
					log.Warn("failed to replace secrets", zap.Error(err))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else {
					agentDeployment.ComposeFile = patchedComposeFile
					agentDeployment.EnvFile = envFile
					agentDeployment.DockerType = util.PtrCopy(deployment.DockerType)
				}
			} else {
				agentDeployment.ReleaseName = *deployment.ReleaseName
				agentDeployment.ChartUrl = *appVersion.ChartUrl
				agentDeployment.ChartVersion = *appVersion.ChartVersion
				if versionValues, err := appVersion.ParsedValuesFile(); err != nil {
					log.Warn("parse error", zap.Error(err))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if deploymentValues, err := deploymentvalues.ParsedValuesFileReplaceSecrets(
					&deployment,
					secrets,
				); err != nil {
					log.Warn("parse error", zap.Error(err))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if merged, err := util.MergeAllRecursive(versionValues, deploymentValues); err != nil {
					log.Warn("merge error", zap.Error(err))
					http.Error(w, fmt.Sprintf("error merging values files: %v", err), http.StatusInternalServerError)
					return
				} else {
					agentDeployment.Values = merged
				}
				if *appVersion.ChartType == types.HelmChartTypeRepository {
					agentDeployment.ChartName = *appVersion.ChartName
				}
				if deployment.HelmOptions != nil {
					agentDeployment.HelmOptions = &api.HelmOptions{
						Timeout:           deployment.HelmOptions.Timeout,
						WaitStrategy:      deployment.HelmOptions.WaitStrategy,
						RollbackOnFailure: deployment.HelmOptions.RollbackOnFailure,
						CleanupOnFailure:  deployment.HelmOptions.CleanupOnFailure,
					}
				}
			}
			agentResource.Deployments = append(agentResource.Deployments, agentDeployment)
		}

		if statusMessage == "OK" {
			RespondJSON(w, agentResource)
		}
	}

	// not in a TX because insertion should not be rolled back when the cleanup fails
	if err := db.CreateDeploymentTargetStatus(ctx, &deploymentTarget.DeploymentTarget, statusMessage); err != nil {
		log.Error("failed to create deployment target status – skipping cleanup of old statuses", zap.Error(err),
			zap.String("deploymentTargetId", deploymentTarget.ID.String()),
			zap.String("statusMessage", statusMessage))
	}
}

func agentPutDeploymentLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.AgentAuthentication.Require(ctx)
		records, err := JsonBody[[]api.DeploymentLogRecord](w, r)
		if err != nil {
			return
		}

		if err := db.ValidateDeploymentLogRecords(ctx, auth.CurrentDeploymentTargetID(), records); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.Error(w, fmt.Sprintf("bad request: %v", err), http.StatusBadRequest)
			} else {
				log.Error("error saving deployment log records", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		if err := db.SaveDeploymentLogRecords(ctx, records); errors.Is(err, apierrors.ErrBadRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else if err != nil {
			log.Error("error saving deployment log records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func agentPutDeploymentTargetLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)
		records, err := JsonBody[[]api.DeploymentTargetLogRecordRequest](w, r)
		if err != nil {
			return
		}

		if err := db.SaveDeploymentTargetLogRecords(ctx, deploymentTarget.ID, records); err != nil {
			log.Error("error saving deployment target log records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func patchProjectName(data map[string]any, deploymentID uuid.UUID) ([]byte, error) {
	if data == nil {
		data = make(map[string]any)
	}
	data["name"] = fmt.Sprintf("distr-%v", deploymentID.String()[:8])
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func agentPostStatusHandler(w http.ResponseWriter, r *http.Request) {
	requestBody, err := JsonBody[api.AgentDeploymentStatus](w, r)
	if err != nil {
		return
	}

	ctx := r.Context()
	log := internalctx.GetLogger(ctx).With(zap.Any("status", requestBody))
	sentry := sentry.GetHubFromContext(ctx)

	deploymentID, err := db.GetDeploymentIDForRevisionID(ctx, requestBody.RevisionID)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			sentry.CaptureException(err)
			log.Error("failed to get deployment ID", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	deploymentTarget := internalctx.GetDeploymentTarget(ctx)
	var deployment types.DeploymentWithLatestRevision
	if i := slices.IndexFunc(
		deploymentTarget.Deployments,
		func(d types.DeploymentWithLatestRevision) bool { return d.ID == deploymentID },
	); i < 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	} else {
		deployment = deploymentTarget.Deployments[i]
	}

	previousStatus, err := db.GetLatestDeploymentRevisionStatus(ctx, deploymentID)
	if err != nil {
		sentry.CaptureException(err)
		log.Error("failed to get latest deployment revision status", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	status := types.DeploymentRevisionStatus{
		DeploymentRevisionID: requestBody.RevisionID,
		Type:                 requestBody.Type,
		Message:              requestBody.Message,
	}

	if err := db.CreateDeploymentRevisionStatus(ctx, &status); err != nil {
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else {
			log.Error("failed to create deployment revision status", zap.Error(err))
			sentry.CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	go func(ctx context.Context) {
		asyncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := notification.SendDeploymentStatusNotifications(
			asyncCtx,
			*deploymentTarget,
			deployment,
			previousStatus,
			status,
		); err != nil {
			sentry.CaptureException(err)
			log.Error("failed to dispatch deployment status notification", zap.Error(err))
		}
	}(context.WithoutCancel(ctx))

	w.WriteHeader(http.StatusOK)
}

func agentPostMetricsHander(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	dt := internalctx.GetDeploymentTarget(ctx)

	metrics, err := JsonBody[api.AgentDeploymentTargetMetrics](w, r)
	if err != nil {
		return
	}
	if err := db.CreateDeploymentTargetMetrics(ctx, &dt.DeploymentTarget, &metrics); err != nil {
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else {
			log.Error("failed to create deployment target metrics – skipping cleanup of old metrics", zap.Error(err),
				zap.Reflect("metrics", metrics))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func queryAuthDeploymentTargetCtxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		targetID, err := uuid.Parse(r.URL.Query().Get("targetId"))
		if err != nil {
			http.Error(w, "targetId is not a valid UUID", http.StatusBadRequest)
			return
		}
		targetSecret := r.URL.Query().Get("targetSecret")

		if agentConnectPerTargetIdRateLimiter.RespondOnLimit(w, r, targetID.String()) {
			return
		} else if deploymentTarget, err := getVerifiedDeploymentTarget(ctx, targetID, targetSecret); err != nil {
			if errors.Is(err, apierrors.ErrUnauthorized) {
				log.Info("agent unauthorized", zap.Error(err), zap.Stringer("deploymentTargetId", targetID))
				http.Error(w, err.Error(), http.StatusUnauthorized)
			} else {
				log.Error("failed to get deployment target from query auth", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		} else {
			ctx = internalctx.WithDeploymentTarget(ctx, deploymentTarget)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func agentManifestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)
		log := internalctx.GetLogger(ctx).With(zap.String("deploymentTargetId", deploymentTarget.ID.String()))
		if org, err := db.GetOrganizationByID(ctx, deploymentTarget.OrganizationID); err != nil {
			log.Error("could not get org for deployment target", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else if manifest, err := agentmanifest.Get(ctx, *deploymentTarget, *org, nil); err != nil {
			log.Error("could not get agent manifest", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			w.Header().Add("Content-Type", "application/yaml")
			if _, err := io.Copy(w, manifest); err != nil {
				log.Warn("writing to client failed", zap.Error(err))
			}
		}
	}
}

func agentAuthDeploymentTargetCtxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.AgentAuthentication.Require(ctx)
		orgId := auth.CurrentOrgID()
		targetId := auth.CurrentDeploymentTargetID()

		if deploymentTarget, err := db.GetDeploymentTarget(ctx, targetId, &orgId); errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusUnauthorized)
		} else if err != nil {
			log.Error("failed to get DeploymentTarget", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if ua := r.UserAgent(); strings.HasPrefix(ua, fmt.Sprintf("%v/", useragent.DistrAgentUserAgent)) {
				reportedVersionName := strings.TrimPrefix(ua, fmt.Sprintf("%v/", useragent.DistrAgentUserAgent))
				if reportedVersion, err := db.GetAgentVersionWithName(ctx, reportedVersionName); err != nil {
					log.Error("could not get reported agent version", zap.Error(err))
					sentry.GetHubFromContext(ctx).CaptureException(err)
				} else if deploymentTarget.ReportedAgentVersionID == nil ||
					reportedVersion.ID != *deploymentTarget.ReportedAgentVersionID {
					if err := db.UpdateDeploymentTargetReportedAgentVersionID(
						ctx, deploymentTarget, reportedVersion.ID); err != nil {
						log.Error("could not update reported agent version", zap.Error(err))
						sentry.GetHubFromContext(ctx).CaptureException(err)
					}
				}
			}
			ctx = internalctx.WithDeploymentTarget(ctx, deploymentTarget)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func getVerifiedDeploymentTarget(
	ctx context.Context,
	targetID uuid.UUID,
	targetSecret string,
) (*types.DeploymentTargetFull, error) {
	if deploymentTarget, err := db.GetDeploymentTarget(ctx, targetID, nil); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return nil, fmt.Errorf("%w: %w", apierrors.ErrUnauthorized, err)
		}
		return nil, fmt.Errorf("failed to get deployment target from DB: %w", err)
	} else if deploymentTarget.AccessKeySalt == nil || deploymentTarget.AccessKeyHash == nil {
		return nil, fmt.Errorf("%w: deployment target does not have key and salt", apierrors.ErrUnauthorized)
	} else if err := security.VerifyAccessKey(
		*deploymentTarget.AccessKeySalt, *deploymentTarget.AccessKeyHash, targetSecret); err != nil {
		return nil, fmt.Errorf("%w: failed to verify access: %w", apierrors.ErrUnauthorized, err)
	} else {
		return deploymentTarget, nil
	}
}

var (
	agentConnectPerTargetIdRateLimiter = httprate.NewRateLimiter(5, time.Minute)
	agentLoginPerTargetIdRateLimiter   = httprate.NewRateLimiter(5, time.Minute)

	rateLimitPerAgent = httprate.Limit(
		// For a 5 second interval, per minute, the agent makes 12 resource calls and 12 status calls for each deployment.
		// Adding 25% margin and assuming that people have at most 10 deployments on a single agent we arrive at
		// (12+10*12)*1.25 = 11*12*1.25 = 11*15
		// also adding 2 for the metric reports
		(11*15)+2,
		1*time.Minute,
		httprate.WithKeyFuncs(middleware.RateLimitCurrentDeploymentTargetIdKeyFunc),
	)
)
