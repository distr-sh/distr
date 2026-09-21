package handlers

import (
	"context"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/api"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// automaticUpdatesAllowed reports whether the organization and the application currently permit
// automatic updates. A deployment can outlive all three conditions, since the vendor may withdraw
// them from the application and the organization may lose the feature with its plan.
func automaticUpdatesAllowed(org *types.Organization, application *types.Application) bool {
	return org.HasFeature(types.FeatureAutoUpdates) &&
		application.AllowAutomaticUpdates &&
		application.VersioningStrategy.AllowsAutomaticUpdates()
}

// triggerAutomaticApplicationUpdates rolls every deployment of the application that has automatic
// updates enabled forward to the newest version it is entitled to.
//
// It only ever moves a deployment forward, which is why archiving a version and narrowing an
// entitlement need no special case: both can only lower the newest version, and the deployment is
// then already ahead of it. That also makes the function safe to call after any change to an
// application, its versions or an entitlement.
//
// The caller has to have written its own change to the application before calling, and to be inside
// a transaction, since the newest version is determined from what the database holds rather than
// from what the caller has in memory.
func triggerAutomaticApplicationUpdates(
	ctx context.Context,
	org *types.Organization,
	applicationID uuid.UUID,
	createdByUserID *uuid.UUID,
) error {
	return updateDeploymentsToLatestVersion(ctx, org, applicationID, nil, createdByUserID)
}

// triggerAutomaticUpdateOfDeployment is triggerAutomaticApplicationUpdates for the single
// deployment automatic updates have just been enabled on.
func triggerAutomaticUpdateOfDeployment(
	ctx context.Context,
	org *types.Organization,
	applicationID uuid.UUID,
	deploymentID uuid.UUID,
	createdByUserID *uuid.UUID,
) error {
	return updateDeploymentsToLatestVersion(ctx, org, applicationID, &deploymentID, createdByUserID)
}

// automaticUpdateError writes the response for a failed automatic update, for a handler whose
// transaction only rolls back on the error it returns.
func automaticUpdateError(ctx context.Context, w http.ResponseWriter, err error) error {
	internalctx.GetLogger(ctx).Warn("could not trigger automatic application updates", zap.Error(err))
	sentry.GetHubFromContext(ctx).CaptureException(err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	return err
}

func updateDeploymentsToLatestVersion(
	ctx context.Context,
	org *types.Organization,
	applicationID uuid.UUID,
	onlyDeploymentID *uuid.UUID,
	createdByUserID *uuid.UUID,
) error {
	// Two transactions creating a version at the same time each miss the other's version, and a
	// revision's created_at is the start of the transaction that inserted it, so the one that
	// started later can make the older of the two versions the latest revision. The lock, and
	// reading the versions and the current revisions only while holding it, keep that out.
	if err := db.LockApplicationExclusive(ctx, applicationID); err != nil {
		return err
	}

	application, err := db.GetApplication(ctx, applicationID, org.ID)
	if err != nil {
		return err
	}
	if !automaticUpdatesAllowed(org, application) {
		return nil
	}

	deployments, err := db.GetDeploymentsWithAutomaticApplicationUpdates(ctx, applicationID)
	if err != nil {
		return err
	}
	if onlyDeploymentID != nil {
		deployments = slices.DeleteFunc(deployments, func(d types.DeploymentWithLatestRevision) bool {
			return d.ID != *onlyDeploymentID
		})
	}
	if len(deployments) == 0 {
		return nil
	}

	// The ordering is taken from all of the application's versions rather than from the ones a
	// single deployment is entitled to, so that every deployment of the application agrees on it.
	compare := types.ApplicationVersionComparator(application.VersioningStrategy, application.Versions)

	byTarget := make(map[uuid.UUID][]types.DeploymentWithLatestRevision)
	for _, deployment := range deployments {
		byTarget[deployment.DeploymentTargetID] = append(byTarget[deployment.DeploymentTargetID], deployment)
	}

	for targetID, deploymentsForTarget := range byTarget {
		target, err := db.GetDeploymentTarget(ctx, targetID, nil, nil)
		if err != nil {
			return err
		}
		secrets, err := db.GetSecretsForDeploymentTarget(ctx, target.DeploymentTarget)
		if err != nil {
			return err
		}
		licenseKeys, err := db.GetLicenseKeysForDeploymentTarget(ctx, target.DeploymentTarget)
		if err != nil {
			return err
		}

		for _, deployment := range deploymentsForTarget {
			if err := updateDeploymentToLatestVersion(
				ctx, application, deployment, compare, secrets, licenseKeys, createdByUserID,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func updateDeploymentToLatestVersion(
	ctx context.Context,
	application *types.Application,
	deployment types.DeploymentWithLatestRevision,
	compare func(a, b types.ApplicationVersion) int,
	secrets []types.SecretWithUpdatedBy,
	licenseKeys []types.LicenseKey,
	createdByUserID *uuid.UUID,
) error {
	log := internalctx.GetLogger(ctx).With(zap.String("deploymentId", deployment.ID.String()))

	candidates, err := entitledApplicationVersions(ctx, application, deployment)
	if err != nil {
		return err
	}
	latest := types.LatestApplicationVersion(compare, candidates)
	if latest == nil {
		return nil
	}
	currentIndex := slices.IndexFunc(application.Versions, func(v types.ApplicationVersion) bool {
		return v.ID == deployment.ApplicationVersionID
	})
	if currentIndex < 0 {
		log.Warn("skipping automatic update because the deployed version does not belong to the application")
		return nil
	}
	if compare(*latest, application.Versions[currentIndex]) <= 0 {
		return nil
	}

	latestWithFiles, err := db.GetApplicationVersion(ctx, latest.ID)
	if err != nil {
		return err
	}

	request := deploymentRequestFromLatestRevision(deployment)
	request.ApplicationVersionID = latest.ID
	request.CreatedByUserAccountID = createdByUserID
	request.Trigger = types.DeploymentRevisionTriggerAutomaticUpdate

	if err := validateAutomaticUpdateValues(&request, latestWithFiles, secrets, licenseKeys); err != nil {
		log.Warn("skipping automatic update because the deployment values do not apply to the new version",
			zap.String("applicationVersionId", latest.ID.String()), zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		return nil
	}
	if err := setDeploymentRequestValuesHash(&request, secrets, licenseKeys); err != nil {
		log.Warn("skipping automatic update because the deployment values could not be rendered", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		return nil
	}
	if _, err := db.CreateDeploymentRevision(ctx, &request); err != nil {
		return err
	}

	log.Info("deployment updated automatically",
		zap.String("applicationVersionId", latest.ID.String()),
		zap.String("applicationVersionName", latest.Name))
	return nil
}

// entitledApplicationVersions are the versions the deployment may be updated to. An entitlement
// without versions entitles the customer to all of them.
func entitledApplicationVersions(
	ctx context.Context,
	application *types.Application,
	deployment types.DeploymentWithLatestRevision,
) ([]types.ApplicationVersion, error) {
	if deployment.ApplicationEntitlementID == nil {
		return application.Versions, nil
	}
	entitlement, err := db.GetApplicationEntitlementByID(ctx, *deployment.ApplicationEntitlementID)
	if err != nil {
		return nil, err
	}
	if len(entitlement.Versions) == 0 {
		return application.Versions, nil
	}
	return entitlement.Versions, nil
}

func validateAutomaticUpdateValues(
	request *api.DeploymentRequest,
	applicationVersion *types.ApplicationVersion,
	secrets []types.SecretWithUpdatedBy,
	licenseKeys []types.LicenseKey,
) error {
	deploymentValues, err := deploymentvalues.ParsedValuesFileReplaceSecrets(request, secrets, licenseKeys)
	if err != nil {
		return err
	}
	applicationVersionValues, err := applicationVersion.ParsedValuesFile()
	if err != nil {
		return err
	}
	if _, err := util.MergeAllRecursive(applicationVersionValues, deploymentValues); err != nil {
		return err
	}
	_, err = deploymentvalues.EnvFileReplaceSecrets(request, secrets, licenseKeys)
	return err
}
