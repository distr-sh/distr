package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/distr-sh/distr/internal/requestlog"
	"github.com/distr-sh/distr/internal/respond"
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

		r.Get("/pre-connect", respond.Handler(preConnectHandler)).
			With(option.Request(ControllerConnectRequest{})).
			With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
		r.Get("/connect", respond.Handler(connectHandler)).
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
			respond.Middleware(controllerAuthDeploymentTargetCtxMiddleware),
			rateLimitPerDeploymentTargetID,
		).Group(func(r chiopenapi.Router) {
			// controller routes, authenticated via token.
			// manifest and resources are read-only and served from the read-only db. The auth
			// middleware above (which may write the reported controller version) stays on the primary.
			r.With(middleware.UseReadonlyDB).Get("/manifest", respond.Handler(controllerManifestHandler))
			r.With(middleware.UseReadonlyDB).Get("/resources", respond.Handler(controllerResourcesHandler))
			r.Post("/status", respond.Handler(controllerPostStatusHandler))
			r.Post("/metrics", respond.Handler(controllerPostMetricsHander))
			r.Post("/deployments/{deploymentId}/metrics", respond.Handler(controllerPostDeploymentMetricsHandler))
			r.Put("/logs", respond.Handler(controllerPutDeploymentLogsHandler))
			r.Put("/deployment-target-logs", respond.Handler(controllerPutDeploymentTargetLogsHandler))
		})
	}
	r.Route("/controller", controllerRoutes)
	// Controllers released before the rename to controller and every running one that has not been
	// reconfigured since read their endpoints from their environment, which points here.
	r.Route("/agent", controllerRoutes)
}

func connectHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)
	secret := r.URL.Query().Get("targetSecret")

	if org, err := db.GetOrganizationWithBranding(ctx, deploymentTarget.OrganizationID); err != nil {
		return nil, fmt.Errorf("could not get organization for deployment target: %w", err)
	} else if manifest, err := controllermanifest.Get(ctx, *deploymentTarget, *org, &secret); err != nil {
		return nil, fmt.Errorf("could not get controller manifest: %w", err)
	} else {
		return respond.Stream("application/yaml", manifest), nil
	}
}

// optionally wraps the connect request in a shell script
func preConnectHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)
	secret := r.URL.Query().Get("targetSecret")

	if org, err := db.GetOrganizationWithBranding(ctx, deploymentTarget.OrganizationID); err != nil {
		return nil, fmt.Errorf("could not get organization for deployment target: %w", err)
	} else if script, err := controllerconnect.GenerateConnectScript(ctx, deploymentTarget.ID, *org, secret); err != nil {
		return nil, fmt.Errorf("could not generate connect script: %w", err)
	} else {
		return respond.Stream("text/plain; charset=utf-8", strings.NewReader(script)), nil
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
		} else if deploymentTarget, err := getVerifiedDeploymentTarget(ctx, parsedTargetID, targetSecret, true); err != nil {
			if errors.Is(err, apierrors.ErrUnauthorized) {
				log.Info("controller unauthorized", zap.Error(err), zap.String("deploymentTargetId", targetID))
				http.Error(w, err.Error(), http.StatusUnauthorized)
			} else {
				log.Error("failed to get deployment target from basic auth", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		} else {
			recordDeploymentTarget(ctx, deploymentTarget)
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

func controllerResourcesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)

	if deployments, err := db.GetDeploymentsForDeploymentTarget(ctx, deploymentTarget.ID); err != nil {
		return nil, fmt.Errorf("failed to get latest Deployment from DB: %w", err)
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
				return nil, fmt.Errorf("failed to get ApplicationVersion from DB: %w", err)
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
					return nil, fmt.Errorf("failed to get ApplicationEntitlement from DB: %w", err)
				} else if entitlement.RegistryURL != nil {
					controllerDeployment.RegistryAuth = map[string]api.ControllerRegistryAuth{
						*entitlement.RegistryURL: {
							Username: string(*entitlement.RegistryUsername),
							Password: string(*entitlement.RegistryPassword),
						},
					}
				}
			}

			if secrets, err := db.GetSecretsForDeploymentTarget(ctx, deploymentTarget.DeploymentTarget); err != nil {
				return nil, fmt.Errorf("failed to get secrets from DB: %w", err)
			} else if licenseKeys, err := db.GetLicenseKeysForDeploymentTarget(
				ctx, deploymentTarget.DeploymentTarget,
			); err != nil {
				return nil, fmt.Errorf("failed to get license keys from DB: %w", err)
			} else if deploymentTarget.Type == types.DeploymentTypeDocker {
				if composeYaml, err := appVersion.ParsedComposeFile(); err != nil {
					return nil, fmt.Errorf("parse error: %w", err)
				} else if patchedComposeFile, err := patchProjectName(composeYaml, deployment.ID); err != nil {
					return nil, fmt.Errorf("failed to patch project name: %w", err)
				} else if envFile, err := deploymentvalues.EnvFileReplaceSecrets(&deployment, secrets, licenseKeys); err != nil {
					return nil, fmt.Errorf("failed to replace secrets: %w", err)
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
					return nil, fmt.Errorf("parse error: %w", err)
				} else if deploymentValues, err := deploymentvalues.ParsedValuesFileReplaceSecrets(
					&deployment,
					secrets,
					licenseKeys,
				); err != nil {
					return nil, fmt.Errorf("parse error: %w", err)
				} else if merged, err := util.MergeAllRecursive(versionValues, deploymentValues); err != nil {
					return nil, fmt.Errorf("error merging values files: %w", err)
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

		return respond.JSON(controllerResource), nil
	}
}

func controllerPutDeploymentLogsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.ControllerAuthentication.Require(ctx)

	if records, resp := decodeJSON[[]api.DeploymentLogRecord](r); resp != nil {
		return resp, nil
	} else {
		records = sanitizeLogRecords(records)

		// Drop records referencing unknown deployment/revision tuples instead of rejecting the
		// whole batch: the controller buffers records for many deployments together, so a single
		// stale reference must not discard the other deployments' valid logs.
		if valid, err := db.FilterValidDeploymentLogRecords(
			ctx, auth.CurrentDeploymentTargetID(), records,
		); err != nil {
			return nil, fmt.Errorf("error filtering valid deployment log records: %w", err)
		} else {
			if len(valid) < len(records) {
				log.Warn("dropping deployment log records referencing unknown deployment revisions",
					zap.Int("dropped", len(records)-len(valid)), zap.Int("total", len(records)))
			}

			logStore := logstore.FromContext(ctx)
			if err := logStore.SaveDeploymentLogRecords(ctx, auth.CurrentOrgID(), valid); err != nil {
				if errors.Is(err, apierrors.ErrBadRequest) {
					return respond.BadRequest(err.Error()), nil
				} else if errors.Is(err, logstore.ErrRateLimitExceeded) {
					return respond.Error(http.StatusTooManyRequests, err.Error()), nil
				} else {
					return nil, fmt.Errorf("error saving deployment log records: %w", err)
				}
			} else {
				return respond.NoContent(), nil
			}
		}
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

func controllerPutDeploymentTargetLogsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)

	if records, resp := decodeJSON[[]api.DeploymentTargetLogRecordRequest](r); resp != nil {
		return resp, nil
	} else if err := logstore.FromContext(ctx).SaveDeploymentTargetLogRecords(
		ctx, deploymentTarget.OrganizationID, deploymentTarget.ID, records,
	); err != nil {
		if errors.Is(err, apierrors.ErrBadRequest) {
			return respond.BadRequest(err.Error()), nil
		} else if errors.Is(err, logstore.ErrRateLimitExceeded) {
			return respond.Error(http.StatusTooManyRequests, err.Error()), nil
		} else {
			return nil, fmt.Errorf("error saving deployment target log records: %w", err)
		}
	} else {
		return respond.NoContent(), nil
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

func controllerPostStatusHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)

	if requestBody, resp := decodeJSON[api.ControllerDeploymentStatus](r); resp != nil {
		return resp, nil
	} else if deploymentID, err := db.GetDeploymentIDForRevisionID(ctx, requestBody.RevisionID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return respond.BadRequest(err.Error()), nil
		} else {
			return nil, fmt.Errorf("failed to get deployment ID: %w", err)
		}
	} else if i := slices.IndexFunc(
		deploymentTarget.Deployments,
		func(d types.DeploymentWithLatestRevision) bool { return d.ID == deploymentID },
	); i < 0 {
		return respond.BadRequest(http.StatusText(http.StatusBadRequest)), nil
	} else if settledStatus, err := db.GetPreviousDeploymentRevisionStatus(ctx, requestBody.RevisionID); err != nil {
		return nil, fmt.Errorf("failed to get previous deployment revision status: %w", err)
	} else {
		log := internalctx.GetLogger(ctx).With(zap.Any("status", requestBody))
		sentry := sentry.GetHubFromContext(ctx)
		deployment := deploymentTarget.Deployments[i]
		previousStatus := deployment.NewestStatus()

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
				return respond.BadRequest(http.StatusText(http.StatusBadRequest)), nil
			} else {
				return nil, fmt.Errorf("failed to update deployment revision status: %w", err)
			}
		} else {
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

			return respond.Status(http.StatusOK), nil
		}
	}
}

func controllerPostMetricsHander(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	dt := internalctx.GetDeploymentTarget(ctx)

	if body, resp := decodeJSON[api.ControllerDeploymentTargetMetricsRequest](r); resp != nil {
		return resp, nil
	} else if previousMetrics, err := db.GetLatestDeploymentTargetMetricsForID(ctx, dt.ID); err != nil {
		return nil, fmt.Errorf("failed to get previous deployment target metrics: %w", err)
	} else {
		metrics := mapping.DeploymentTargetMetricsRequestToInternal(dt.ID, body)

		if err := db.CreateDeploymentTargetMetrics(ctx, &metrics); err != nil {
			if errors.Is(err, apierrors.ErrConflict) {
				return respond.BadRequest(http.StatusText(http.StatusBadRequest)), nil
			} else {
				return nil, fmt.Errorf("failed to create deployment target metrics: %w", err)
			}
		} else {
			go func(ctx context.Context) {
				asyncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				if err := notification.SendDeploymentTargetMetricsNotifications(
					asyncCtx, *dt, previousMetrics, metrics,
				); err != nil {
					sentry.GetHubFromContext(asyncCtx).CaptureException(err)
					log.Error("send metrics alerts failed", zap.Error(err))
				}
			}(context.WithoutCancel(ctx))

			return respond.Status(http.StatusOK), nil
		}
	}
}

func controllerPostDeploymentMetricsHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	dt := internalctx.GetDeploymentTarget(ctx)

	if deploymentID, err := uuid.Parse(r.PathValue("deploymentId")); err != nil {
		return respond.BadRequest("deploymentId is not a valid UUID"), nil
	} else if !slices.ContainsFunc(
		dt.Deployments,
		func(d types.DeploymentWithLatestRevision) bool { return d.ID == deploymentID },
	) {
		return respond.NotFound(http.StatusText(http.StatusNotFound)), nil
	} else if body, resp := decodeJSON[api.ControllerDeploymentResourceMetricsRequest](r); resp != nil {
		return resp, nil
	} else {
		metrics := mapping.DeploymentResourceMetricsRequestToInternal(deploymentID, body)

		if err := db.CreateDeploymentMetrics(ctx, &metrics); err != nil {
			return nil, fmt.Errorf("failed to create deployment metrics: %w", err)
		} else {
			return respond.Status(http.StatusOK), nil
		}
	}
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
		} else if deploymentTarget, err := getVerifiedDeploymentTarget(ctx, targetID, targetSecret, false); err != nil {
			if errors.Is(err, apierrors.ErrUnauthorized) {
				log.Info("controller unauthorized", zap.Error(err), zap.Stringer("deploymentTargetId", targetID))
				http.Error(w, err.Error(), http.StatusUnauthorized)
			} else {
				log.Error("failed to get deployment target from query auth", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		} else {
			recordDeploymentTarget(ctx, deploymentTarget)
			ctx = internalctx.WithDeploymentTarget(ctx, deploymentTarget)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}

func recordDeploymentTarget(ctx context.Context, deploymentTarget *types.DeploymentTargetFull) {
	requestlog.FromContext(ctx).Update(func(f *requestlog.Fields) {
		f.OrganizationID = &deploymentTarget.OrganizationID
		f.DeploymentTargetID = &deploymentTarget.ID
		if deploymentTarget.CustomerOrganization != nil {
			f.CustomerOrganizationID = &deploymentTarget.CustomerOrganization.ID
		}
	})
}

func controllerManifestHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)

	if org, err := db.GetOrganizationWithBranding(ctx, deploymentTarget.OrganizationID); err != nil {
		return nil, fmt.Errorf("could not get org for deployment target %v: %w", deploymentTarget.ID, err)
	} else if manifest, err := controllermanifest.Get(ctx, *deploymentTarget, *org, nil); err != nil {
		return nil, fmt.Errorf("could not get controller manifest for deployment target %v: %w", deploymentTarget.ID, err)
	} else {
		return respond.Stream("application/yaml", manifest), nil
	}
}

func controllerAuthDeploymentTargetCtxMiddleware(r *http.Request) (*http.Request, respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.ControllerAuthentication.Require(ctx)
	orgId := auth.CurrentOrgID()

	if deploymentTarget, err := db.GetDeploymentTarget(
		ctx, auth.CurrentDeploymentTargetID(), &orgId, nil,
	); errors.Is(err, apierrors.ErrNotFound) {
		return nil, respond.Unauthorized("deployment target not found"), nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get deployment target: %w", err)
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
		recordDeploymentTarget(ctx, deploymentTarget)
		return r.WithContext(internalctx.WithDeploymentTarget(ctx, deploymentTarget)), nil, nil
	}
}

func getVerifiedDeploymentTarget(
	ctx context.Context,
	targetID uuid.UUID,
	targetSecret string,
	promotePendingAccessKey bool,
) (*types.DeploymentTargetFull, error) {
	deploymentTarget, err := db.GetDeploymentTarget(ctx, targetID, nil, nil)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return nil, fmt.Errorf("%w: %w", apierrors.ErrUnauthorized, err)
		}
		return nil, fmt.Errorf("failed to get deployment target from DB: %w", err)
	}

	pending, err := security.VerifyDeploymentTargetAccessKey(deploymentTarget.DeploymentTarget, targetSecret)
	if errors.Is(err, security.ErrNoAccessKey) {
		return nil, fmt.Errorf("%w: deployment target does not have key and salt", apierrors.ErrUnauthorized)
	} else if err != nil {
		return nil, fmt.Errorf("%w: failed to verify access: %w", apierrors.ErrUnauthorized, err)
	}

	if pending && promotePendingAccessKey {
		if err := db.PromoteDeploymentTargetPendingAccessKey(
			ctx, deploymentTarget.ID, *deploymentTarget.PendingAccessKeyHash,
		); errors.Is(err, apierrors.ErrNotFound) {
			return nil, fmt.Errorf("%w: pending access key was replaced", apierrors.ErrUnauthorized)
		} else if err != nil {
			return nil, err
		}
		internalctx.GetLogger(ctx).Info("pending access key promoted",
			zap.Stringer("deploymentTargetId", deploymentTarget.ID))
	}

	return deploymentTarget, nil
}
