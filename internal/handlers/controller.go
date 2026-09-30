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
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authjwt"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/controllerclient/useragent"
	"github.com/distr-sh/distr/internal/controllerconnect"
	"github.com/distr-sh/distr/internal/controllermanifest"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/logstore"
	"github.com/distr-sh/distr/internal/mapping"
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

func ControllerRouter(r chiopenapi.Router) {
	rateLimitPerDeploymentTargetID := httprate.LimitBy(
		500,
		1*time.Minute,
		middleware.RateLimitCurrentDeploymentTargetIdKeyFunc,
	)

	// pre-connect and connect only read (org + deployment target) to render the controller manifest. The
	// query-auth middleware runs first on the primary, then reads are served from the read-only db.
	r.With(queryAuthDeploymentTargetCtxMiddleware, middleware.UseReadonlyDB).Group(func(r chiopenapi.Router) {
		r.WithOptions(option.GroupTags("Controllers"))

		type ControllerConnectRequest struct {
			TargetID     uuid.UUID `query:"targetId"`
			TargetSecret string    `query:"targetSecret"`
		}

		r.Get("/pre-connect", preConnectHandler()).
			With(option.Request(ControllerConnectRequest{})).
			With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
		r.Get("/connect", connectHandler()).
			With(option.Request(ControllerConnectRequest{})).
			With(option.Response(http.StatusOK, map[string]any{}, option.ContentType("application/yaml")))
	})

	controllerRoutes := func(r chiopenapi.Router) {
		r.WithOptions(option.GroupHidden(true))
		// controller login (from basic auth to token)
		r.Post("/login", controllerLoginHandler())

		r.With(
			auth.ControllerAuthentication.Middleware,
			middleware.SetSentryUserFromControllerAuth,
			controllerAuthDeploymentTargetCtxMiddleware,
			rateLimitPerDeploymentTargetID,
		).Group(func(r chiopenapi.Router) {
			// controller routes, authenticated via token.
			// manifest and resources are read-only and served from the read-only db. The auth
			// middleware above (which may write the reported controller version) stays on the primary.
			r.With(middleware.UseReadonlyDB).Get("/manifest", controllerManifestHandler())
			r.With(middleware.UseReadonlyDB).Get("/resources", controllerResourcesHandler)
			r.Post("/status", controllerPostStatusHandler)
			r.Post("/metrics", controllerPostMetricsHander)
			r.Post("/deployments/{deploymentId}/metrics", controllerPostDeploymentMetricsHandler)
			r.Put("/logs", controllerPutDeploymentLogsHandler())
			r.Put("/deployment-target-logs", controllerPutDeploymentTargetLogsHandler())
		})
	}
	r.Route("/controller", controllerRoutes)
	// Controllers released before the rename to controller and every running one that has not been
	// reconfigured since read their endpoints from their environment, which points here.
	r.Route("/agent", controllerRoutes)
}

func connectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)

		org, err := db.GetOrganizationWithBranding(ctx, deploymentTarget.OrganizationID)
		if err != nil {
			log.Error("could not get organization for deployment target", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		secret := r.URL.Query().Get("targetSecret")
		if manifest, err := controllermanifest.Get(ctx, *deploymentTarget, *org, &secret); err != nil {
			log.Error("could not get controller manifest", zap.Error(err))
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

		org, err := db.GetOrganizationWithBranding(ctx, deploymentTarget.OrganizationID)
		if err != nil {
			log.Error("could not get organization for deployment target", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		secret := r.URL.Query().Get("targetSecret")
		script, err := controllerconnect.GenerateConnectScript(ctx, deploymentTarget.ID, *org, secret)
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

func controllerLoginHandler() func(w http.ResponseWriter, r *http.Request) {
	limiter := httprate.NewRateLimiter(5, time.Minute)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		if targetID, targetSecret, ok := r.BasicAuth(); !ok {
			log.Info("invalid Basic Auth")
			w.WriteHeader(http.StatusUnauthorized)
		} else if parsedTargetID, err := uuid.Parse(targetID); err != nil {
			http.Error(w, "targetId is not a valid UUID", http.StatusBadRequest)
		} else if limiter.RespondOnLimit(w, r, targetID) {
			return
		} else if deploymentTarget, err := getVerifiedDeploymentTarget(ctx, parsedTargetID, targetSecret); err != nil {
			if errors.Is(err, apierrors.ErrUnauthorized) {
				log.Info("controller unauthorized", zap.Error(err), zap.String("deploymentTargetId", targetID))
				http.Error(w, err.Error(), http.StatusUnauthorized)
			} else {
				log.Error("failed to get deployment target from basic auth", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		} else {
			// TODO maybe even randomize token valid duration
			if _, token, err := authjwt.GenerateControllerTokenValidFor(
				deploymentTarget.ID, deploymentTarget.OrganizationID, env.ControllerTokenMaxValidDuration()); err != nil {
				log.Error("failed to create controller token", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				if err := json.NewEncoder(w).Encode(api.AuthLoginResponse{Token: token}); err != nil {
					log.Error("failed to write response", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		}
	}
}

func controllerResourcesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)
	log := internalctx.GetLogger(ctx).With(zap.String("deploymentTargetId", deploymentTarget.ID.String()))

	deployments, err := db.GetDeploymentsForDeploymentTarget(ctx, deploymentTarget.ID)
	if err != nil {
		log.Error("failed to get latest Deployment from DB", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		controllerResource := api.ControllerResource{
			Version:               deploymentTarget.ControllerVersion,
			MetricsEnabled:        deploymentTarget.MetricsEnabled,
			DeploymentLogsEnabled: deploymentTarget.DeploymentLogsEnabled,
			DeploymentLogsAfter:   deploymentTarget.DeploymentLogsAfter,
		}
		if deploymentTarget.Namespace != nil {
			controllerResource.Namespace = *deploymentTarget.Namespace
		}

		for _, deployment := range deployments {
			appVersion, err := db.GetApplicationVersion(ctx, deployment.ApplicationVersionID)
			if err != nil {
				log.Error("failed to get ApplicationVersion from DB", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			controllerDeployment := api.ControllerDeployment{
				ID:         deployment.ID,
				RevisionID: deployment.DeploymentRevisionID,
				//nolint:staticcheck // kept for controllers that don't read ControllerResource.DeploymentLogsEnabled
				LogsEnabled:        deploymentTarget.DeploymentLogsEnabled,
				ForceRestart:       deployment.ForceRestart,
				IgnoreRevisionSkew: deployment.IgnoreRevisionSkew,
			}

			if deployment.ApplicationEntitlementID != nil {
				if entitlement, err := db.GetApplicationEntitlementByID(ctx, *deployment.ApplicationEntitlementID); err != nil {
					log.Error("failed to get ApplicationEntitlement from DB", zap.Error(err))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if entitlement.RegistryURL != nil {
					controllerDeployment.RegistryAuth = map[string]api.ControllerRegistryAuth{
						*entitlement.RegistryURL: {
							Username: string(*entitlement.RegistryUsername),
							Password: string(*entitlement.RegistryPassword),
						},
					}
				}
			}

			var secrets []types.SecretWithUpdatedBy
			if secrets, err = db.GetSecretsForDeploymentTarget(ctx, deploymentTarget.DeploymentTarget); err != nil {
				log.Error("failed to get secrets from DB", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			licenseKeys, err := db.GetLicenseKeysForDeploymentTarget(ctx, deploymentTarget.DeploymentTarget)
			if err != nil {
				log.Error("failed to get license keys from DB", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
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
				} else if envFile, err := deploymentvalues.EnvFileReplaceSecrets(&deployment, secrets, licenseKeys); err != nil {
					log.Warn("failed to replace secrets", zap.Error(err))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else {
					controllerDeployment.ComposeFile = patchedComposeFile
					controllerDeployment.EnvFile = envFile
					controllerDeployment.DockerType = util.PtrCopy(deployment.DockerType)
					controllerDeployment.ImageCleanupEnabled = deploymentTarget.ImageCleanupEnabled
				}
			} else {
				controllerDeployment.ReleaseName = *deployment.ReleaseName
				controllerDeployment.ChartUrl = *appVersion.ChartUrl
				controllerDeployment.ChartVersion = *appVersion.ChartVersion
				if versionValues, err := appVersion.ParsedValuesFile(); err != nil {
					log.Warn("parse error", zap.Error(err))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if deploymentValues, err := deploymentvalues.ParsedValuesFileReplaceSecrets(
					&deployment,
					secrets,
					licenseKeys,
				); err != nil {
					log.Warn("parse error", zap.Error(err))
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if merged, err := util.MergeAllRecursive(versionValues, deploymentValues); err != nil {
					log.Warn("merge error", zap.Error(err))
					http.Error(w, fmt.Sprintf("error merging values files: %v", err), http.StatusInternalServerError)
					return
				} else {
					controllerDeployment.Values = merged
				}
				if *appVersion.ChartType == types.HelmChartTypeRepository {
					controllerDeployment.ChartName = *appVersion.ChartName
				}
				if deployment.HelmOptions != nil {
					controllerDeployment.HelmOptions = &api.HelmOptions{
						Timeout:           deployment.HelmOptions.Timeout,
						WaitStrategy:      deployment.HelmOptions.WaitStrategy,
						RollbackOnFailure: deployment.HelmOptions.RollbackOnFailure,
						CleanupOnFailure:  deployment.HelmOptions.CleanupOnFailure,
						ForceConflicts:    deployment.HelmOptions.ForceConflicts,
					}
				}
			}
			controllerResource.Deployments = append(controllerResource.Deployments, controllerDeployment)
		}

		RespondJSON(w, controllerResource)
	}
}

func controllerPutDeploymentLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.ControllerAuthentication.Require(ctx)
		records, err := JsonBody[[]api.DeploymentLogRecord](w, r)
		if err != nil {
			return
		}

		records = sanitizeLogRecords(records)

		// Drop records referencing unknown deployment/revision tuples instead of rejecting the
		// whole batch: the controller buffers records for many deployments together, so a single
		// stale reference must not discard the other deployments' valid logs.
		if valid, err := db.FilterValidDeploymentLogRecords(ctx, auth.CurrentDeploymentTargetID(), records); err != nil {
			log.Error("error filtering valid deployment log records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else {
			if len(valid) < len(records) {
				log.Warn("dropping deployment log records referencing unknown deployment revisions",
					zap.Int("dropped", len(records)-len(valid)), zap.Int("total", len(records)))
			}
			records = valid
		}

		logStore := logstore.FromContext(ctx)
		if err := logStore.SaveDeploymentLogRecords(ctx, auth.CurrentOrgID(), records); err != nil {
			if errors.Is(err, apierrors.ErrBadRequest) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, logstore.ErrRateLimitExceeded) {
				http.Error(w, err.Error(), http.StatusTooManyRequests)
				return
			}
			log.Error("error saving deployment log records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func sanitizeLogRecords(records []api.DeploymentLogRecord) []api.DeploymentLogRecord {
	filteredRecords := make([]api.DeploymentLogRecord, 0, len(records))
	for _, record := range records {
		record.Body = strings.ReplaceAll(record.Body, "\x00", "")
		if record.Body != "" {
			filteredRecords = append(filteredRecords, record)
		}
	}
	return filteredRecords
}

func controllerPutDeploymentTargetLogsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)
		records, err := JsonBody[[]api.DeploymentTargetLogRecordRequest](w, r)
		if err != nil {
			return
		}

		logStore := logstore.FromContext(ctx)
		if err := logStore.SaveDeploymentTargetLogRecords(
			ctx, deploymentTarget.OrganizationID, deploymentTarget.ID, records,
		); err != nil {
			if errors.Is(err, apierrors.ErrBadRequest) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if errors.Is(err, logstore.ErrRateLimitExceeded) {
				http.Error(w, err.Error(), http.StatusTooManyRequests)
				return
			}

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

func controllerPostStatusHandler(w http.ResponseWriter, r *http.Request) {
	requestBody, err := JsonBody[api.ControllerDeploymentStatus](w, r)
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

	previousStatus := deployment.NewestStatus()
	settledStatus, err := db.GetLatestSettledDeploymentRevisionStatus(ctx, deploymentID)
	if err != nil {
		sentry.CaptureException(err)
		log.Error("failed to get latest settled deployment revision status", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var status *types.DeploymentRevisionStatus
	if err := db.RunTx(ctx, func(ctx context.Context) error {
		var err error
		if status, err = db.UpdateDeploymentRevisionStatus(
			ctx, requestBody.RevisionID, requestBody.Type, requestBody.Message,
		); err != nil {
			return err
		}
		if status.Type.IsApplied() {
			return db.UpdateDeploymentCurrentRevision(ctx, status.DeploymentRevisionID)
		}
		return nil
	}); err != nil {
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else {
			log.Error("failed to update deployment revision status", zap.Error(err))
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
			settledStatus,
			*status,
		); err != nil {
			sentry.CaptureException(err)
			log.Error("failed to dispatch deployment status notification", zap.Error(err))
		}
	}(context.WithoutCancel(ctx))

	w.WriteHeader(http.StatusOK)
}

func controllerPostMetricsHander(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	dt := internalctx.GetDeploymentTarget(ctx)

	body, err := JsonBody[api.ControllerDeploymentTargetMetricsRequest](w, r)
	if err != nil {
		return
	}

	previousMetrics, err := db.GetLatestDeploymentTargetMetricsForID(ctx, dt.ID)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to get previous deployment target metrics", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	metrics := mapping.DeploymentTargetMetricsRequestToInternal(dt.ID, body)

	if err := db.CreateDeploymentTargetMetrics(ctx, &metrics); err != nil {
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to create deployment target metrics", zap.Error(err), zap.Any("metrics", body))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	go func(ctx context.Context) {
		asyncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := notification.SendDeploymentTargetMetricsNotifications(asyncCtx, *dt, previousMetrics, metrics); err != nil {
			sentry.GetHubFromContext(asyncCtx).CaptureException(err)
			log.Error("send metrics alerts failed", zap.Error(err))
		}
	}(context.WithoutCancel(ctx))

	w.WriteHeader(http.StatusOK)
}

func controllerPostDeploymentMetricsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	deploymentID, err := uuid.Parse(r.PathValue("deploymentId"))
	if err != nil {
		http.Error(w, "deploymentId is not a valid UUID", http.StatusBadRequest)
		return
	}

	dt := internalctx.GetDeploymentTarget(ctx)
	if !slices.ContainsFunc(
		dt.Deployments,
		func(d types.DeploymentWithLatestRevision) bool { return d.ID == deploymentID },
	) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	body, err := JsonBody[api.ControllerDeploymentResourceMetricsRequest](w, r)
	if err != nil {
		return
	}

	metrics := mapping.DeploymentResourceMetricsRequestToInternal(deploymentID, body)

	if err := db.CreateDeploymentMetrics(ctx, &metrics); err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to create deployment metrics", zap.Error(err), zap.Any("metrics", body))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func queryAuthDeploymentTargetCtxMiddleware(next http.Handler) http.Handler {
	limiter := httprate.NewRateLimiter(5, time.Minute)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		targetID, err := uuid.Parse(r.URL.Query().Get("targetId"))
		if err != nil {
			http.Error(w, "targetId is not a valid UUID", http.StatusBadRequest)
			return
		}
		targetSecret := r.URL.Query().Get("targetSecret")

		if limiter.RespondOnLimit(w, r, targetID.String()) {
			return
		} else if deploymentTarget, err := getVerifiedDeploymentTarget(ctx, targetID, targetSecret); err != nil {
			if errors.Is(err, apierrors.ErrUnauthorized) {
				log.Info("controller unauthorized", zap.Error(err), zap.Stringer("deploymentTargetId", targetID))
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

func controllerManifestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		deploymentTarget := internalctx.GetDeploymentTarget(ctx)
		log := internalctx.GetLogger(ctx).With(zap.String("deploymentTargetId", deploymentTarget.ID.String()))
		if org, err := db.GetOrganizationWithBranding(ctx, deploymentTarget.OrganizationID); err != nil {
			log.Error("could not get org for deployment target", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else if manifest, err := controllermanifest.Get(ctx, *deploymentTarget, *org, nil); err != nil {
			log.Error("could not get controller manifest", zap.Error(err))
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

func controllerAuthDeploymentTargetCtxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.ControllerAuthentication.Require(ctx)
		orgId := auth.CurrentOrgID()
		targetId := auth.CurrentDeploymentTargetID()

		deploymentTarget, err := db.GetDeploymentTarget(ctx, targetId, &orgId, nil)
		if errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusUnauthorized)
		} else if err != nil {
			log.Error("failed to get DeploymentTarget", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if reportedVersionName, ok := useragent.ReportedVersion(r.UserAgent()); ok {
				if reportedVersion, err := db.GetControllerVersionWithName(ctx, reportedVersionName); err != nil {
					log.Error("could not get reported controller version", zap.Error(err))
					sentry.GetHubFromContext(ctx).CaptureException(err)
				} else if deploymentTarget.ReportedControllerVersionID == nil ||
					reportedVersion.ID != *deploymentTarget.ReportedControllerVersionID {
					if err := db.UpdateDeploymentTargetReportedControllerVersionID(
						ctx, deploymentTarget, reportedVersion.ID); err != nil {
						log.Error("could not update reported controller version", zap.Error(err))
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
	if deploymentTarget, err := db.GetDeploymentTarget(ctx, targetID, nil, nil); err != nil {
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
