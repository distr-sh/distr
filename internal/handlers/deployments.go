package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func DeploymentsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Deployments"))
	r.Use(middleware.RequireOrgAndRole)
	r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Put("/", putDeployment).
		With(option.Description("Create or update a deployment")).
		With(option.Request(api.DeploymentRequest{}))
	r.With(deploymentMiddleware).Route("/{deploymentId}", func(r chiopenapi.Router) {
		type DeploymentIDRequest struct {
			DeploymentID uuid.UUID `path:"deploymentId"`
		}

		type DeploymentTimeseriesRequest struct {
			DeploymentIDRequest
			TimeseriesRequest
		}

		type ResourceRequest struct {
			Resource string `query:"resource"`
		}

		r.Get("/status", getDeploymentStatus).
			With(option.Description("Get deployment status")).
			With(option.Request(DeploymentTimeseriesRequest{})).
			With(option.Response(http.StatusOK, []api.DeploymentRevisionStatus{}))
		r.Get("/status/export", exportDeploymentStatusHandler()).
			With(option.Description("Export deployment status")).
			With(option.Request(DeploymentIDRequest{})).
			With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
		r.Get("/logs", getDeploymentLogsHandler()).
			With(option.Description("Get deployment logs")).
			With(option.Request(struct {
				DeploymentTimeseriesRequest
				ResourceRequest
			}{})).
			With(option.Response(http.StatusOK, []api.DeploymentLogRecord{}))
		r.Get("/logs/resources", getDeploymentLogsResourcesHandler()).
			With(option.Description("Get deployment log resources")).
			With(option.Request(DeploymentIDRequest{})).
			With(option.Response(http.StatusOK, []string{}))
		r.Get("/logs/export", exportDeploymentLogsHandler()).
			With(option.Description("Export deployment logs")).
			With(option.Request(struct {
				DeploymentIDRequest
				ResourceRequest
			}{})).
			With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
			r.Patch("/", patchDeploymentHandler()).
				With(option.Description("Partially update a deployment")).
				With(option.Request(struct {
					DeploymentIDRequest
					api.PatchDeploymentRequest
				}{})).
				With(option.Response(http.StatusOK, types.Deployment{}))
			r.Delete("/", deleteDeploymentHandler()).
				With(option.Description("Delete a deployment")).
				With(option.Request(DeploymentIDRequest{}))
		})
	})
}

func putDeployment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	deploymentRequest, err := JsonBody[api.DeploymentRequest](w, r)
	if err != nil {
		return
	}

	_ = db.RunTx(ctx, func(ctx context.Context) error {
		if err := validateDeploymentRequest(ctx, w, deploymentRequest); err != nil {
			return err
		}

		if deploymentRequest.DeploymentID == nil {
			if err = db.CreateDeployment(ctx, &deploymentRequest); errors.Is(err, apierrors.ErrConflict) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return err
			} else if err != nil {
				log.Warn("could not create deployment", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		} else {
			authInfo := auth.Authentication.Require(ctx)
			deployment, err := db.GetDeployment(
				ctx,
				*deploymentRequest.DeploymentID,
				authInfo.CurrentUserID(),
				*authInfo.CurrentOrgID(),
				authInfo.CurrentCustomerOrgID(),
			)
			if err != nil {
				log.Warn("could not get deployment", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}

			if deployment.ApplicationLicenseID == nil && deploymentRequest.ApplicationLicenseID != nil {
				deployment.ApplicationLicenseID = deploymentRequest.ApplicationLicenseID
				if err := db.UpdateDeploymentLicense(ctx, deployment); err != nil {
					log.Warn("could not set license for deployment", zap.Error(err))
					sentry.GetHubFromContext(ctx).CaptureException(err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return err
				}
			}
		}

		if _, err := db.CreateDeploymentRevision(ctx, &deploymentRequest); err != nil {
			log.Warn("could not create deployment revision", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		// TODO: We might need to send a proper deployment object back, but not sure yet what it looks like
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

func patchDeploymentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		deployment := internalctx.GetDeployment(ctx)
		req, err := JsonBody[api.PatchDeploymentRequest](w, r)
		if err != nil {
			return
		}

		needsUpdate := false

		if req.LogsEnabled != nil && *req.LogsEnabled != deployment.LogsEnabled {
			deployment.LogsEnabled = *req.LogsEnabled
			needsUpdate = true
		}

		if needsUpdate {
			if err := db.UpdateDeployment(ctx, deployment); err != nil {
				log.Warn("deployment update failed", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}

		RespondJSON(w, deployment)
	}
}

func deleteDeploymentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)
		orgId := *auth.CurrentOrgID()
		deployment := internalctx.GetDeployment(ctx)
		_ = db.RunTx(ctx, func(ctx context.Context) error {
			target, err := db.GetDeploymentTargetForDeploymentID(ctx, deployment.ID)
			if err != nil {
				log.Warn("could not get DeploymentTarget", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
			if target.OrganizationID != orgId || !isDeploymentTargetVisible(auth, target.DeploymentTarget) {
				http.NotFound(w, r)
				return apierrors.ErrNotFound
			}

			if err := db.DeleteDeploymentWithID(ctx, deployment.ID); err != nil {
				log.Warn("could not delete Deployment", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}

			return nil
		})
	}
}

func validateDeploymentRequest(
	ctx context.Context,
	w http.ResponseWriter,
	request api.DeploymentRequest,
) error {
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	orgId := *auth.CurrentOrgID()

	var license *types.ApplicationLicense
	var app *types.Application
	var version *types.ApplicationVersion
	var target *types.DeploymentTargetFull
	var secrets []types.SecretWithUpdatedBy

	org := auth.CurrentOrg()
	var err error

	if app, err = db.GetApplicationForApplicationVersionID(ctx, request.ApplicationVersionID, orgId); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return badRequestError(w, "Application does not exist")
		} else {
			log.Warn("could not get Application", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return err
		}
	}

	if version, err = db.GetApplicationVersion(ctx, request.ApplicationVersionID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return badRequestError(w, "ApplicationVersion does not exist")
		} else {
			log.Warn("could not get ApplicationVersion", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return err
		}
	}

	if target, err = db.GetDeploymentTarget(ctx, request.DeploymentTargetID, &orgId); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return badRequestError(w, "DeploymentTarget does not exist")
		} else {
			log.Warn("could not get DeploymentTarget", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return err
		}
	}

	if secrets, err = db.GetSecretsForDeploymentTarget(ctx, target.DeploymentTarget); err != nil {
		log.Warn("could not get Secrets", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return err
	}

	var existingDeployment *types.DeploymentWithLatestRevision
	if request.DeploymentID != nil {
		for _, d := range target.Deployments {
			if d.ID == *request.DeploymentID {
				existingDeployment = &d
				break
			}
		}
		if existingDeployment == nil {
			return badRequestError(w, "DeploymentTarget doesn't have Deployment with the specified ID")
		}
	}

	if existingDeployment != nil {
		if request.ApplicationLicenseID == nil {
			if existingDeployment.ApplicationLicenseID != nil {
				request.ApplicationLicenseID = existingDeployment.ApplicationLicenseID
			}
		} else if existingDeployment.ApplicationLicenseID == nil {
			// Allow setting a license once when the existing deployment has no license but the request provides one.
		} else if *request.ApplicationLicenseID != *existingDeployment.ApplicationLicenseID {
			return badRequestError(w, "can not update license")
		}

		if existingDeployment.ApplicationID != app.ID {
			return badRequestError(w, "can not change application of existing deployment")
		}
	}

	if org.HasFeature(types.FeatureLicensing) {
		if request.ApplicationLicenseID != nil {
			if license, err = db.GetApplicationLicenseByID(ctx, *request.ApplicationLicenseID); err != nil {
				if errors.Is(err, apierrors.ErrNotFound) {
					return licenseNotFoundError(w)
				} else {
					log.Error("could not get ApplicationLicense", zap.Error(err))
					sentry.GetHubFromContext(ctx).CaptureException(err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return err
				}
			}
		} else if auth.CurrentCustomerOrgID() != nil {
			if licenses, err := db.GetApplicationLicensesWithOrganizationID(ctx, orgId, nil); err != nil {
				log.Error("could not get ApplicationLicense", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return err
			} else if len(licenses) > 0 {
				// license ID is required for customer but optional for vendor
				return badRequestError(w, "applicationLicenseId is required")
			}
		}
	} else if request.ApplicationLicenseID != nil {
		return badRequestError(w, "unexpected applicationLicenseId")
	}

	if err = validateDeploymentRequestLicense(ctx, w, request, license, app, target, existingDeployment); err != nil {
		return err
	} else if err = validateDeploymentRequestDeploymentType(w, target, app); err != nil {
		return err
	} else if err = validateDeploymentRequestDeploymentTarget(ctx, w, request, target); err != nil {
		return err
	} else if err = validateDeploymentRequestValues(w, request, version, secrets); err != nil {
		return err
	} else {
		return nil
	}
}

func badRequestError(w http.ResponseWriter, msg string) error {
	http.Error(w, msg, http.StatusBadRequest)
	return errors.New(msg)
}

func licenseNotFoundError(w http.ResponseWriter) error {
	return badRequestError(w, "license does not exist")
}

func invalidLicenseError(w http.ResponseWriter) error {
	return badRequestError(w, "invalid license")
}

func validateDeploymentRequestLicense(
	ctx context.Context,
	w http.ResponseWriter,
	request api.DeploymentRequest,
	license *types.ApplicationLicense,
	app *types.Application,
	target *types.DeploymentTargetFull,
	deployment *types.DeploymentWithLatestRevision,
) error {
	if license != nil {
		auth := auth.Authentication.Require(ctx)

		if license.OrganizationID != *auth.CurrentOrgID() {
			return licenseNotFoundError(w)
		}
		if license.CustomerOrganizationID == nil {
			return invalidLicenseError(w)
		}
		if auth.CurrentCustomerOrgID() != nil && *license.CustomerOrganizationID != *auth.CurrentCustomerOrgID() {
			return licenseNotFoundError(w)
		}
		if target.CustomerOrganizationID == nil || *target.CustomerOrganizationID != *license.CustomerOrganizationID {
			return invalidLicenseError(w)
		}
		if len(license.Versions) > 0 && !license.HasVersionWithID(request.ApplicationVersionID) {
			return invalidLicenseError(w)
		}
		if app.ID != license.ApplicationID {
			return invalidLicenseError(w)
		}
		if deployment != nil && deployment.ApplicationID != license.ApplicationID {
			return badRequestError(w, "license and deployment have applicationId mismatch")
		}
	}
	return nil
}

func validateDeploymentRequestDeploymentType(
	w http.ResponseWriter,
	target *types.DeploymentTargetFull,
	application *types.Application,
) error {
	if target.Type != application.Type {
		return badRequestError(w, "application and deployment target must have the same type")
	}
	return nil
}

func validateDeploymentRequestDeploymentTarget(
	ctx context.Context,
	w http.ResponseWriter,
	request api.DeploymentRequest,
	target *types.DeploymentTargetFull,
) error {
	auth := auth.Authentication.Require(ctx)

	if !isDeploymentTargetVisible(auth, target.DeploymentTarget) {
		err := errors.New("DeploymentTarget not found")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	if request.DeploymentID == nil && len(target.Deployments) > 0 {
		if err := target.AgentVersion.CheckMultiDeploymentSupported(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
	}

	if request.IgnoreRevisionSkew && target.Type != types.DeploymentTypeKubernetes {
		return badRequestError(w, "IgnoreRevisionSkew is only supported for Kubernetes deployments")
	}

	return nil
}

func validateDeploymentRequestValues(
	w http.ResponseWriter,
	deploymentRequest api.DeploymentRequest,
	appVersion *types.ApplicationVersion,
	secrets []types.SecretWithUpdatedBy,
) error {
	if deploymentValues, err := deploymentvalues.ParsedValuesFileReplaceSecrets(&deploymentRequest, secrets); err != nil {
		return badRequestError(w, fmt.Sprintf("invalid values: %v", err.Error()))
	} else if appVersionValues, err := appVersion.ParsedValuesFile(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	} else if _, err := util.MergeAllRecursive(appVersionValues, deploymentValues); err != nil {
		return badRequestError(w, fmt.Sprintf("values cannot be merged with base: %v", err))
	} else if _, err := deploymentvalues.EnvFileReplaceSecrets(&deploymentRequest, secrets); err != nil {
		return badRequestError(w, fmt.Sprintf("invalid env file: %v", err.Error()))
	}
	return nil
}

func getDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	deployment := internalctx.GetDeployment(ctx)
	limit, err := QueryParam(r, "limit", strconv.Atoi, Max(100))
	if errors.Is(err, ErrParamNotDefined) {
		limit = 25
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	before, err := QueryParam(r, "before", ParseTimeFunc(time.RFC3339Nano))
	if err != nil && !errors.Is(err, ErrParamNotDefined) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	after, err := QueryParam(r, "after", ParseTimeFunc(time.RFC3339Nano))
	if err != nil && !errors.Is(err, ErrParamNotDefined) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if deploymentStatus, err := db.GetDeploymentRevisionStatus(ctx, deployment.ID, limit, before, after); err != nil {
		internalctx.GetLogger(ctx).Error("failed to get deploymentstatus", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.List(deploymentStatus, mapping.DeploymentRevisionStatusToAPI))
	}
}

func exportDeploymentStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		deployment := internalctx.GetDeployment(ctx)
		authInfo := auth.Authentication.Require(ctx)
		org := authInfo.CurrentOrg()
		limit := int(subscription.GetLogExportRowsLimit(org.SubscriptionType))

		filename := fmt.Sprintf("%s_deployment_status.log", time.Now().Format("2006-01-02"))

		SetFileDownloadHeaders(w, filename)

		err := db.GetDeploymentRevisionStatusForExport(
			ctx, deployment.ID, limit,
			func(record types.DeploymentRevisionStatus) error {
				_, err := fmt.Fprintf(w, "[%s] [%s] %s\n",
					record.CreatedAt.Format(time.RFC3339),
					record.Type,
					record.Message)
				return err
			},
		)
		if err != nil {
			log.Error("failed to export status records", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			// Note: If headers were already sent, we can't send error response
			return
		}
	}
}

func deploymentMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		auth := auth.Authentication.Require(ctx)
		deploymentId, err := uuid.Parse(r.PathValue("deploymentId"))
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if deployment, err := db.GetDeployment(
			ctx,
			deploymentId,
			auth.CurrentUserID(),
			*auth.CurrentOrgID(),
			auth.CurrentCustomerOrgID(),
		); errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
		} else if err != nil {
			internalctx.GetLogger(ctx).Error("failed to get deployment", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			ctx = internalctx.WithDeployment(ctx, deployment)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	})
}
