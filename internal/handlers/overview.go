package handlers

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

const overviewAccessTokenExpiryWindow = 7 * 24 * time.Hour

func OverviewRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireCustomer)
	r.With(middleware.CustomerFeatureMiddleware(types.CustomerOrganizationFeatureDeploymentTargets)).
		Get("/deployment-targets", getOverviewDeploymentTargets)
	r.With(middleware.CustomerFeatureMiddleware(types.CustomerOrganizationFeatureArtifacts)).
		Get("/artifacts", getOverviewArtifacts)
	r.Get("/checklist", getOverviewChecklist)
	r.With(middleware.LicensingFeatureFlagEnabledMiddleware, middleware.UseReadonlyDB).
		Get("/license", getOverviewLicense)
	r.With(middleware.ProFeature, middleware.AlertsCustomerFeatureMiddleware, middleware.UseReadonlyDB).
		Get("/alerts", getOverviewAlerts)
	r.Get("/access-tokens", getOverviewAccessTokens)
	r.With(
		middleware.ProFeature,
		middleware.CustomerFeatureMiddleware(types.CustomerOrganizationFeatureSupportBundles),
	).Get("/support-bundles", getOverviewSupportBundles)
	r.With(middleware.UseReadonlyDB).Get("/activity", getOverviewActivity)
}

func getOverviewDeploymentTargets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID, customerOrgID := *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()

	targets, err := db.GetDeploymentTargetsByScope(ctx, orgID, &customerOrgID)
	if err != nil {
		respondInternalError(w, r, err, "failed to get deployment targets")
		return
	}
	deployments, err := db.GetOverviewDeployments(ctx, orgID, customerOrgID)
	if err != nil {
		respondInternalError(w, r, err, "failed to get deployments")
		return
	}
	applications, err := db.GetApplicationsByOrgID(ctx, orgID)
	if err != nil {
		respondInternalError(w, r, err, "failed to get applications")
		return
	}
	var entitlements []types.ApplicationEntitlement
	if entitled, err := applicationEntitlementsApply(ctx); err != nil {
		respondInternalError(w, r, err, "failed to check for application entitlements")
		return
	} else if entitled {
		if entitlements, err = db.GetApplicationEntitlementsWithCustomerOrganizationID(
			ctx, customerOrgID, orgID, nil,
		); err != nil {
			respondInternalError(w, r, err, "failed to get application entitlements")
			return
		} else if entitlements == nil {
			entitlements = []types.ApplicationEntitlement{}
		}
	}

	now := time.Now()
	deploymentsByTarget := make(map[uuid.UUID][]api.OverviewDeployment)
	for _, d := range deployments {
		var latest *types.ApplicationVersion
		updateAvailable := false
		if i := slices.IndexFunc(applications, func(a types.Application) bool { return a.ID == d.Application.ID }); i >= 0 {
			latest = types.LatestDeploymentApplicationVersion(applications[i], d.ApplicationEntitlementID, entitlements, now)
			updateAvailable = types.ApplicationUpdateAvailable(applications[i], d.ApplicationVersionID, latest)
		}
		deploymentsByTarget[d.DeploymentTargetID] = append(
			deploymentsByTarget[d.DeploymentTargetID],
			mapping.OverviewDeploymentToAPI(d, latest, updateAvailable),
		)
	}

	result := make([]api.OverviewDeploymentTarget, len(targets))
	for i, target := range targets {
		result[i] = mapping.OverviewDeploymentTargetToAPI(target, deploymentsByTarget[target.ID])
	}
	RespondJSON(w, result)
}

func getOverviewArtifacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID, customerOrgID := *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()

	artifacts, err := visibleArtifacts(ctx)
	if err != nil {
		respondInternalError(w, r, err, "failed to get artifacts")
		return
	}
	artifactIDs := make([]uuid.UUID, len(artifacts))
	for i, a := range artifacts {
		artifactIDs[i] = a.ID
	}
	pulls, err := db.GetLatestArtifactPullsOfCustomerOrganization(ctx, orgID, customerOrgID, artifactIDs)
	if err != nil {
		respondInternalError(w, r, err, "failed to get artifact pulls")
		return
	}

	licensing := auth.CurrentOrg().HasFeature(types.FeatureLicensing)
	response := api.OverviewArtifactsResponse{Artifacts: []api.OverviewArtifact{}, HasPulled: len(pulls) > 0}
	for _, artifact := range artifacts {
		if artifact.HiddenInCustomerOverview {
			continue
		}
		versions, err := customerArtifactVersions(ctx, artifact.ID, customerOrgID, licensing)
		if err != nil {
			respondInternalError(w, r, err, "failed to get versions for artifact")
			return
		}
		latest, latestTag := types.LatestArtifactTag(versions)
		var pull *types.CustomerArtifactPull
		pullIsLatest := false
		if i := slices.IndexFunc(pulls, func(p types.CustomerArtifactPull) bool {
			return p.ArtifactID == artifact.ID
		}); i >= 0 {
			pull = &pulls[i]
			pullIsLatest = types.ArtifactPullIsLatest(versions, latest, pull.VersionName)
		}
		response.Artifacts = append(response.Artifacts,
			mapping.OverviewArtifactToAPI(artifact, latestTag, pull, pullIsLatest))
	}
	RespondJSON(w, response)
}

func getOverviewChecklist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if checklist, err := db.GetOverviewChecklist(
		ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID(), auth.CurrentUserID(),
	); err != nil {
		respondInternalError(w, r, err, "failed to get overview checklist")
	} else {
		RespondJSON(w, mapping.OverviewChecklistToAPI(*checklist))
	}
}

func getOverviewLicense(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if license, err := db.GetOverviewLicense(ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()); err != nil {
		respondInternalError(w, r, err, "failed to get license")
	} else if license == nil {
		w.WriteHeader(http.StatusNoContent)
	} else {
		RespondJSON(w, mapping.OverviewLicenseToAPI(*license))
	}
}

func getOverviewAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if summary, err := db.GetOverviewAlertSummary(ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()); err != nil {
		respondInternalError(w, r, err, "failed to get alert summary")
	} else {
		RespondJSON(w, mapping.OverviewAlertSummaryToAPI(*summary))
	}
}

func getOverviewAccessTokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	tokens, err := db.GetAccessTokens(ctx, auth.CurrentUserID(), *auth.CurrentOrgID())
	if err != nil {
		respondInternalError(w, r, err, "failed to get access tokens")
		return
	}
	now := time.Now()
	threshold := now.Add(overviewAccessTokenExpiryWindow)
	result := []api.OverviewAccessToken{}
	for _, token := range tokens {
		if expiresAt := token.NextExpiresAt(now); expiresAt != nil && expiresAt.Before(threshold) {
			result = append(result, mapping.OverviewAccessTokenToAPI(token, *expiresAt))
		}
	}
	slices.SortFunc(result, func(a, b api.OverviewAccessToken) int { return a.ExpiresAt.Compare(b.ExpiresAt) })
	RespondJSON(w, result)
}

func getOverviewSupportBundles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if bundles, err := db.GetOverviewSupportBundles(ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()); err != nil {
		respondInternalError(w, r, err, "failed to get support bundles")
	} else {
		RespondJSON(w, mapping.List(bundles, mapping.OverviewSupportBundleToAPI))
	}
}

func getOverviewActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	org, customerOrg := auth.CurrentOrg(), auth.CurrentCustomerOrg()

	filter := types.OverviewActivityFilter{
		Deployments: customerOrg.HasFeature(types.CustomerOrganizationFeatureDeploymentTargets),
		Artifacts:   customerOrg.HasFeature(types.CustomerOrganizationFeatureArtifacts),
		Licensing:   org.HasFeature(types.FeatureLicensing),
		SupportBundles: org.SubscriptionType.IsPro() &&
			customerOrg.HasFeature(types.CustomerOrganizationFeatureSupportBundles),
	}
	var err error
	if filter.EntitledApplicationsOnly, err = applicationEntitlementsApply(ctx); err != nil {
		respondInternalError(w, r, err, "failed to check for application entitlements")
		return
	}
	if filter.Licensing {
		if filter.EntitledArtifactsOnly, err = db.HasAnyArtifactEntitlement(ctx, org.ID); err != nil {
			respondInternalError(w, r, err, "failed to check for artifact entitlements")
			return
		}
	}

	if activity, err := db.GetOverviewActivity(ctx, org.ID, customerOrg.ID, filter); err != nil {
		respondInternalError(w, r, err, "failed to get overview activity")
	} else {
		RespondJSON(w, mapping.List(activity, mapping.OverviewActivityToAPI))
	}
}

// applicationEntitlementsApply reports whether application entitlements restrict the current customer,
// which they do only with licensing enabled and once the vendor has created any.
func applicationEntitlementsApply(ctx context.Context) (bool, error) {
	auth := auth.Authentication.Require(ctx)
	if !auth.CurrentOrg().HasFeature(types.FeatureLicensing) {
		return false, nil
	}
	return db.HasAnyApplicationEntitlement(ctx, *auth.CurrentOrgID())
}
