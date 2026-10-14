package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func DeploymentsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Deployments"))
	r.Use(middleware.RequireOrgAndRole)
	r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).
		Put("/", respond.Handler(putDeployment)).
		With(option.Description("Create or update a deployment")).
		With(option.Request(api.DeploymentRequest{}))
	r.With(respond.Middleware(deploymentMiddleware)).Route("/{deploymentId}", func(r chiopenapi.Router) {
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

		r.Get("/revisions", respond.Handler(getDeploymentRevisions)).
			With(option.Description("Get deployment revisions")).
			With(option.Request(DeploymentIDRequest{})).
			With(option.Response(http.StatusOK, []api.DeploymentRevisionResponse{}))
		// These are read-only, controller-pushed timeseries that are safe to serve from the read-only db.
		r.With(middleware.UseReadonlyDB).Group(func(r chiopenapi.Router) {
			r.Get("/metrics", respond.Handler(getDeploymentMetrics)).
				With(option.Description("Get the latest resource metrics reported for a deployment")).
				With(option.Request(DeploymentIDRequest{})).
				With(option.Response(http.StatusOK, api.DeploymentResourceMetrics{}))
			r.Get("/logs", respond.Handler(getDeploymentLogsHandler)).
				With(option.Description("Get deployment logs")).
				With(option.Request(struct {
					DeploymentTimeseriesRequest
					ResourceRequest
				}{})).
				With(option.Response(http.StatusOK, []api.DeploymentLogRecord{}))
			r.Get("/logs/resources", respond.Handler(getDeploymentLogsResourcesHandler)).
				With(option.Description("Get deployment log resources")).
				With(option.Request(DeploymentIDRequest{})).
				With(option.Response(http.StatusOK, api.DeploymentLogRecordResourcesResponse{}))
			r.Get("/logs/export", exportDeploymentLogsHandler()).
				With(option.Description("Export deployment logs")).
				With(option.Request(struct {
					DeploymentIDRequest
					ResourceRequest
					TimeseriesRangeRequest
				}{})).
				With(option.Response(http.StatusOK, nil, option.ContentType("text/plain")))
		})
		r.With(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
			r.Delete("/", respond.Handler(deleteDeployment)).
				With(option.Description("Delete a deployment")).
				With(option.Request(DeploymentIDRequest{}))
			r.With(middleware.AutoUpdatesFeatureMiddleware).
				Patch("/", respond.Handler(patchDeployment)).
				With(option.Description("Partially update a deployment")).
				With(option.Request(struct {
					DeploymentIDRequest
					api.PatchDeploymentRequest
				}{}))
		})
	})
}

func putDeployment(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	if request, resp := decodeJSON[api.DeploymentRequest](r); resp != nil {
		return resp, nil
	} else if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		return saveDeployment(ctx, request)
	}); resp != nil || err != nil {
		return resp, err
	} else {
		// TODO: We might need to send a proper deployment object back, but not sure yet what it looks like
		return respond.NoContent(), nil
	}
}

func saveDeployment(ctx context.Context, request api.DeploymentRequest) (respond.Responder, error) {
	request.CreatedByUserAccountID = new(auth.Authentication.Require(ctx).CurrentUserID())
	request.Trigger = types.DeploymentRevisionTriggerUser

	if validationResult, resp, err := validateDeploymentRequest(ctx, request); resp != nil || err != nil {
		return resp, err
	} else if err := setDeploymentRequestValuesHash(
		&request, validationResult.Secrets, validationResult.LicenseKeys,
	); err != nil {
		return deploymentValuesError(err, "invalid deployment values")
	} else if resp, err := createOrUpdateDeployment(
		ctx, &request, validationResult.AutomaticApplicationUpdatesEnabled,
	); resp != nil || err != nil {
		return resp, err
	} else if _, err := db.CreateDeploymentRevision(ctx, &request); err != nil {
		return nil, fmt.Errorf("failed to create deployment revision: %w", err)
	} else {
		return nil, nil
	}
}

func createOrUpdateDeployment(
	ctx context.Context,
	request *api.DeploymentRequest,
	automaticUpdates bool,
) (respond.Responder, error) {
	authInfo := auth.Authentication.Require(ctx)
	if request.DeploymentID == nil {
		request.AutomaticApplicationUpdatesEnabled = &automaticUpdates
		if err := db.CreateDeployment(ctx, request); errors.Is(err, apierrors.ErrConflict) {
			return respond.BadRequest("release name must be unique per deployment target"), nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to create deployment: %w", err)
		} else {
			return nil, nil
		}
	} else if deployment, err := db.GetDeployment(
		ctx,
		*request.DeploymentID,
		authInfo.CurrentUserID(),
		*authInfo.CurrentOrgID(),
		authInfo.CurrentCustomerOrgID(),
		authInfo.CurrentPartnerOrgID(),
	); err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	} else {
		if deployment.ApplicationEntitlementID == nil && request.ApplicationEntitlementID != nil {
			deployment.ApplicationEntitlementID = request.ApplicationEntitlementID
			if err := db.UpdateDeploymentEntitlement(ctx, deployment); err != nil {
				return nil, fmt.Errorf("failed to set entitlement for deployment: %w", err)
			}
		}
		if requested := request.AutomaticApplicationUpdatesEnabled; requested != nil &&
			deployment.AutomaticApplicationUpdatesEnabled != *requested {
			deployment.AutomaticApplicationUpdatesEnabled = *requested
			if err := db.UpdateDeploymentAutomaticApplicationUpdates(ctx, deployment); err != nil {
				return nil, fmt.Errorf("failed to set automatic updates for deployment: %w", err)
			}
		}
		return nil, nil
	}
}

func patchDeployment(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deployment := internalctx.GetDeployment(ctx)
	if patch, resp := decodeJSON[api.PatchDeploymentRequest](r); resp != nil {
		return resp, nil
	} else if patch.AutomaticApplicationUpdatesEnabled == nil {
		return respond.NoContent(), nil
	} else if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		return setDeploymentAutomaticUpdates(ctx, deployment, *patch.AutomaticApplicationUpdatesEnabled)
	}); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.NoContent(), nil
	}
}

func setDeploymentAutomaticUpdates(
	ctx context.Context,
	deployment *types.Deployment,
	enabled bool,
) (respond.Responder, error) {
	authInfo := auth.Authentication.Require(ctx)
	if target, err := db.GetDeploymentTargetForDeploymentID(ctx, deployment.ID); err != nil {
		return nil, fmt.Errorf("failed to get deployment target: %w", err)
	} else if target.OrganizationID != *authInfo.CurrentOrgID() || !isDeploymentTargetVisible(ctx, target) {
		return deploymentNotFound, nil
	} else if index := slices.IndexFunc(target.Deployments, func(d types.DeploymentWithLatestRevision) bool {
		return d.ID == deployment.ID
	}); index < 0 {
		return deploymentNotFound, nil
	} else if application, err := db.GetApplication(
		ctx, target.Deployments[index].Application.ID, target.OrganizationID,
	); err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	} else if enabled && !automaticUpdatesAllowed(authInfo.CurrentOrg(), application) {
		return respond.BadRequest("this application does not allow automatic updates"), nil
	} else {
		if deployment.AutomaticApplicationUpdatesEnabled != enabled {
			deployment.AutomaticApplicationUpdatesEnabled = enabled
			if err := db.UpdateDeploymentAutomaticApplicationUpdates(ctx, deployment); err != nil {
				return nil, fmt.Errorf("failed to update deployment: %w", err)
			}
		}

		// Enabling has to catch the deployment up, otherwise it would stay behind until the next
		// version is created.
		if enabled {
			if err := triggerAutomaticUpdateOfDeployment(
				ctx, authInfo.CurrentOrg(), application.ID, deployment.ID, new(authInfo.CurrentUserID()),
			); err != nil {
				return nil, fmt.Errorf("failed to trigger automatic update: %w", err)
			}
		}
		return nil, nil
	}
}

func deleteDeployment(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	deployment := internalctx.GetDeployment(ctx)
	if resp, err := runTx(ctx, func(ctx context.Context) (respond.Responder, error) {
		if target, err := db.GetDeploymentTargetForDeploymentID(ctx, deployment.ID); err != nil {
			return nil, fmt.Errorf("failed to get deployment target: %w", err)
		} else if target.OrganizationID != *auth.CurrentOrgID() || !isDeploymentTargetVisible(ctx, target) {
			return deploymentNotFound, nil
		} else if err := db.DeleteDeploymentWithID(ctx, deployment.ID); err != nil {
			return nil, fmt.Errorf("failed to delete deployment: %w", err)
		} else {
			return nil, nil
		}
	}); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.NoContent(), nil
	}
}

func validateDeploymentRequest(
	ctx context.Context,
	request api.DeploymentRequest,
) (*deploymentRequestValidationResult, respond.Responder, error) {
	auth := auth.Authentication.Require(ctx)
	orgId := *auth.CurrentOrgID()

	var entitlement *types.ApplicationEntitlement
	var app *types.Application
	var version *types.ApplicationVersion
	var target *types.DeploymentTargetFull
	var secrets []types.SecretWithUpdatedBy
	var licenseKeys []types.LicenseKey

	org := auth.CurrentOrg()
	var resp respond.Responder
	var err error

	if app, err = db.GetApplicationForApplicationVersionID(
		ctx, request.ApplicationVersionID, orgId,
	); errors.Is(err, apierrors.ErrNotFound) {
		return nil, respond.BadRequest("Application does not exist"), nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get application: %w", err)
	}

	if version, err = db.GetApplicationVersion(ctx, request.ApplicationVersionID); errors.Is(err, apierrors.ErrNotFound) {
		return nil, respond.BadRequest("ApplicationVersion does not exist"), nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get application version: %w", err)
	}

	if target, err = db.GetDeploymentTarget(
		ctx,
		request.DeploymentTargetID,
		&orgId,
		auth.CurrentPartnerOrgID(),
	); errors.Is(err, apierrors.ErrNotFound) {
		return nil, respond.BadRequest("DeploymentTarget does not exist"), nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get deployment target: %w", err)
	}

	if secrets, err = db.GetSecretsForDeploymentTarget(ctx, target.DeploymentTarget); err != nil {
		return nil, nil, fmt.Errorf("failed to get secrets: %w", err)
	}

	if licenseKeys, err = db.GetLicenseKeysForDeploymentTarget(ctx, target.DeploymentTarget); err != nil {
		return nil, nil, fmt.Errorf("failed to get license keys: %w", err)
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
			return nil, respond.BadRequest("DeploymentTarget doesn't have Deployment with the specified ID"), nil
		}
	}

	if existingDeployment != nil {
		if request.ApplicationEntitlementID == nil {
			if existingDeployment.ApplicationEntitlementID != nil {
				request.ApplicationEntitlementID = existingDeployment.ApplicationEntitlementID
			}
		} else if existingDeployment.ApplicationEntitlementID == nil {
			// Allow setting an entitlement once when the existing deployment has no entitlement but the request provides one.
		} else if *request.ApplicationEntitlementID != *existingDeployment.ApplicationEntitlementID {
			return nil, respond.BadRequest("can not update entitlement"), nil
		}

		if existingDeployment.Application.ID != app.ID {
			return nil, respond.BadRequest("can not change application of existing deployment"), nil
		}
	}

	if entitlement, resp, err = resolveDeploymentEntitlement(ctx, request, org); resp != nil || err != nil {
		return nil, resp, err
	}

	automaticUpdates := util.PtrDerefOrDefault(request.AutomaticApplicationUpdatesEnabled)
	if request.AutomaticApplicationUpdatesEnabled == nil && existingDeployment != nil {
		automaticUpdates = existingDeployment.AutomaticApplicationUpdatesEnabled &&
			automaticUpdatesAllowed(org, app)
	}

	if resp := validateDeploymentRequestEntitlement(
		ctx, request, entitlement, app, target, existingDeployment,
	); resp != nil {
		return nil, resp, nil
	} else if resp := validateDeploymentRequestDeploymentType(target, app); resp != nil {
		return nil, resp, nil
	} else if resp := validateDeploymentRequestDeploymentTarget(ctx, request, target); resp != nil {
		return nil, resp, nil
	} else if resp, err := validateDeploymentRequestValues(
		request, version, secrets, licenseKeys,
	); resp != nil || err != nil {
		return nil, resp, err
	} else if resp := validateAutomaticApplicationUpdates(
		request.ApplicationVersionID, automaticUpdates, entitlement, app, org,
	); resp != nil {
		return nil, resp, nil
	} else {
		return &deploymentRequestValidationResult{
			Target:                             target.DeploymentTarget,
			Secrets:                            secrets,
			LicenseKeys:                        licenseKeys,
			AutomaticApplicationUpdatesEnabled: automaticUpdates,
		}, nil, nil
	}
}

// validateAutomaticApplicationUpdates keeps a deployment with automatic updates on the newest
// version the customer is entitled to, so that the next update does not immediately supersede the
// revision this request creates.
func validateAutomaticApplicationUpdates(
	applicationVersionID uuid.UUID,
	enabled bool,
	entitlement *types.ApplicationEntitlement,
	app *types.Application,
	org *types.Organization,
) respond.Responder {
	if !enabled {
		return nil
	}
	if !org.HasFeature(types.FeatureAutoUpdates) {
		return respond.BadRequest("automatic updates are not enabled for this organization")
	}
	if !automaticUpdatesAllowed(org, app) {
		return respond.BadRequest("this application does not allow automatic updates")
	}
	versions := app.Versions
	if entitlement != nil && len(entitlement.Versions) > 0 {
		versions = entitlement.Versions
	}
	latest := types.LatestApplicationVersion(
		types.ApplicationVersionComparator(app.VersioningStrategy, app.Versions),
		versions,
	)
	if latest == nil || latest.ID != applicationVersionID {
		return respond.BadRequest("a deployment with automatic updates must use the latest application version")
	}
	return nil
}

type deploymentRequestValidationResult struct {
	Target                             types.DeploymentTarget
	Secrets                            []types.SecretWithUpdatedBy
	LicenseKeys                        []types.LicenseKey
	AutomaticApplicationUpdatesEnabled bool
}

func setDeploymentRequestValuesHash(
	request *api.DeploymentRequest,
	secrets []types.SecretWithUpdatedBy,
	licenseKeys []types.LicenseKey,
) error {
	hash, err := deploymentvalues.RenderAndHash(request, secrets, licenseKeys)
	if err != nil {
		return err
	}
	request.ValuesHash = hash[:]
	return nil
}

func resolveDeploymentEntitlement(
	ctx context.Context,
	request api.DeploymentRequest,
	org *types.Organization,
) (*types.ApplicationEntitlement, respond.Responder, error) {
	if !org.HasFeature(types.FeatureLicensing) {
		if request.ApplicationEntitlementID != nil {
			return nil, respond.BadRequest("unexpected applicationEntitlementId"), nil
		}
		return nil, nil, nil
	}

	authInfo := auth.Authentication.Require(ctx)

	if request.ApplicationEntitlementID != nil {
		if entitlement, err := db.GetApplicationEntitlementByID(
			ctx, *request.ApplicationEntitlementID,
		); errors.Is(err, apierrors.ErrNotFound) {
			return nil, entitlementDoesNotExist, nil
		} else if err != nil {
			return nil, nil, fmt.Errorf("failed to get application entitlement: %w", err)
		} else {
			return entitlement, nil, nil
		}
	}

	if authInfo.CurrentCustomerOrgID() != nil {
		if entitlements, err := db.GetApplicationEntitlementsWithOrganizationID(
			ctx, *authInfo.CurrentOrgID(), nil,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to get application entitlements: %w", err)
		} else if len(entitlements) > 0 {
			// entitlement ID is required for customer but optional for vendor
			return nil, respond.BadRequest("applicationEntitlementId is required"), nil
		}
	}

	return nil, nil, nil
}

var (
	entitlementDoesNotExist = respond.BadRequest("entitlement does not exist")
	invalidEntitlement      = respond.BadRequest("invalid entitlement")
)

func validateDeploymentRequestEntitlement(
	ctx context.Context,
	request api.DeploymentRequest,
	entitlement *types.ApplicationEntitlement,
	app *types.Application,
	target *types.DeploymentTargetFull,
	deployment *types.DeploymentWithLatestRevision,
) respond.Responder {
	if entitlement != nil {
		auth := auth.Authentication.Require(ctx)

		if entitlement.OrganizationID != *auth.CurrentOrgID() {
			return entitlementDoesNotExist
		}
		if entitlement.CustomerOrganizationID == nil {
			return invalidEntitlement
		}
		if auth.CurrentCustomerOrgID() != nil && *entitlement.CustomerOrganizationID != *auth.CurrentCustomerOrgID() {
			return entitlementDoesNotExist
		}
		if target.CustomerOrganizationID == nil || *target.CustomerOrganizationID != *entitlement.CustomerOrganizationID {
			return invalidEntitlement
		}
		if len(entitlement.Versions) > 0 && !entitlement.HasVersionWithID(request.ApplicationVersionID) {
			return invalidEntitlement
		}
		if app.ID != entitlement.ApplicationID {
			return invalidEntitlement
		}
		if deployment != nil && deployment.Application.ID != entitlement.ApplicationID {
			return respond.BadRequest("entitlement and deployment have applicationId mismatch")
		}
	}
	return nil
}

func validateDeploymentRequestDeploymentType(
	target *types.DeploymentTargetFull,
	application *types.Application,
) respond.Responder {
	if target.Type != application.Type {
		return respond.BadRequest("application and deployment target must have the same type")
	}
	return nil
}

func validateDeploymentRequestDeploymentTarget(
	ctx context.Context,
	request api.DeploymentRequest,
	target *types.DeploymentTargetFull,
) respond.Responder {
	if !isDeploymentTargetVisible(ctx, target) {
		return respond.BadRequest("DeploymentTarget not found")
	}

	if request.DeploymentID == nil && len(target.Deployments) > 0 {
		if err := target.ControllerVersion.CheckMultiDeploymentSupported(); err != nil {
			return respond.BadRequest(err.Error())
		}
	}

	if request.IgnoreRevisionSkew && target.Type != types.DeploymentTypeKubernetes {
		return respond.BadRequest("IgnoreRevisionSkew is only supported for Kubernetes deployments")
	}

	return nil
}

func validateDeploymentRequestValues(
	deploymentRequest api.DeploymentRequest,
	appVersion *types.ApplicationVersion,
	secrets []types.SecretWithUpdatedBy,
	licenseKeys []types.LicenseKey,
) (respond.Responder, error) {
	if deploymentValues, err := deploymentvalues.ParsedValuesFileReplaceSecrets(
		&deploymentRequest, secrets, licenseKeys,
	); err != nil {
		return deploymentValuesError(err, "invalid values")
	} else if appVersionValues, err := appVersion.ParsedValuesFile(); err != nil {
		return nil, fmt.Errorf("failed to parse values file of application version: %w", err)
	} else if _, err := util.MergeAllRecursive(appVersionValues, deploymentValues); err != nil {
		return respond.BadRequest(fmt.Sprintf("values cannot be merged with base: %v", err)), nil
	} else if _, err := deploymentvalues.EnvFileReplaceSecrets(&deploymentRequest, secrets, licenseKeys); err != nil {
		return deploymentValuesError(err, "invalid env file")
	} else {
		return nil, nil
	}
}

// deploymentValuesError answers an invalid template with 400 and treats every other error as
// unexpected.
func deploymentValuesError(err error, clientMsg string) (respond.Responder, error) {
	if errors.Is(err, deploymentvalues.ErrInvalidTemplate) {
		return respond.BadRequest(fmt.Sprintf("%s: %v", clientMsg, err)), nil
	} else {
		return nil, fmt.Errorf("%s: %w", clientMsg, err)
	}
}

func getDeploymentRevisions(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deployment := internalctx.GetDeployment(ctx)
	authInfo := auth.Authentication.Require(ctx)
	if revisions, err := db.GetDeploymentRevisions(ctx, deployment.ID); err != nil {
		return nil, fmt.Errorf("failed to get deployment revisions: %w", err)
	} else {
		return respond.JSON(mapping.List(revisions, mapping.DeploymentRevisionToAPI(
			authInfo.CurrentCustomerOrgID(),
			authInfo.CurrentPartnerOrgID(),
		))), nil
	}
}

func getDeploymentMetrics(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	deployment := internalctx.GetDeployment(ctx)
	if metrics, err := db.GetLatestDeploymentMetricsForDeploymentID(ctx, deployment.ID); err != nil {
		return nil, fmt.Errorf("failed to get deployment resource metrics: %w", err)
	} else if metrics == nil {
		return respond.NoContent(), nil
	} else {
		return respond.JSON(mapping.DeploymentMetricsToAPI(*metrics)), nil
	}
}

var deploymentNotFound = respond.NotFound("deployment not found")

func deploymentMiddleware(r *http.Request) (*http.Request, respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if deploymentId, err := uuid.Parse(r.PathValue("deploymentId")); err != nil {
		return nil, deploymentNotFound, nil
	} else if deployment, err := db.GetDeployment(
		ctx,
		deploymentId,
		auth.CurrentUserID(),
		*auth.CurrentOrgID(),
		auth.CurrentCustomerOrgID(),
		auth.CurrentPartnerOrgID(),
	); errors.Is(err, apierrors.ErrNotFound) {
		return nil, deploymentNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get deployment: %w", err)
	} else {
		return r.WithContext(internalctx.WithDeployment(ctx, deployment)), nil, nil
	}
}
