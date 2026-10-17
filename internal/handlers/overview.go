package handlers

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
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
		Get("/deployment-targets", respond.Handler(getOverviewDeploymentTargets))
	r.With(middleware.CustomerFeatureMiddleware(types.CustomerOrganizationFeatureArtifacts)).
		Get("/artifacts", respond.Handler(getOverviewArtifacts))
	r.Get("/checklist", respond.Handler(getOverviewChecklist))
	r.With(middleware.LicensingFeatureFlagEnabledMiddleware, middleware.UseReadonlyDB).
		Get("/license", respond.Handler(getOverviewLicense))
	r.With(middleware.ProFeature, middleware.AlertsCustomerFeatureMiddleware, middleware.UseReadonlyDB).
		Get("/alerts", respond.Handler(getOverviewAlerts))
	r.Get("/access-tokens", respond.Handler(getOverviewAccessTokens))
	r.With(
		middleware.ProFeature,
		middleware.CustomerFeatureMiddleware(types.CustomerOrganizationFeatureSupportBundles),
	).Get("/support-bundles", respond.Handler(getOverviewSupportBundles))
	r.With(middleware.UseReadonlyDB).Get("/activity", respond.Handler(getOverviewActivity))
}

func getOverviewDeploymentTargets(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID, customerOrgID := *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()

	if targets, err := db.GetDeploymentTargetsByScope(ctx, orgID, &customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to get deployment targets: %w", err)
	} else if deployments, err := db.GetOverviewDeployments(ctx, orgID, customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	} else if applications, err := db.GetApplicationsByOrgID(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	} else if entitlements, err := overviewApplicationEntitlements(ctx, orgID, customerOrgID); err != nil {
		return nil, err
	} else {
		now := time.Now()
		deploymentsByTarget := make(map[uuid.UUID][]api.OverviewDeployment)
		for _, d := range deployments {
			var latest *types.ApplicationVersion
			updateAvailable := false
			if i := slices.IndexFunc(applications, func(a types.Application) bool {
				return a.ID == d.Application.ID
			}); i >= 0 {
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
		return respond.JSON(result), nil
	}
}

// overviewApplicationEntitlements returns the application entitlements of the customer organization, or nil
// when entitlements do not restrict it.
func overviewApplicationEntitlements(
	ctx context.Context,
	orgID, customerOrgID uuid.UUID,
) ([]types.ApplicationEntitlement, error) {
	if entitled, err := applicationEntitlementsApply(ctx); err != nil {
		return nil, fmt.Errorf("failed to check for application entitlements: %w", err)
	} else if !entitled {
		return nil, nil
	} else if entitlements, err := db.GetApplicationEntitlementsWithCustomerOrganizationID(
		ctx, customerOrgID, orgID, nil,
	); err != nil {
		return nil, fmt.Errorf("failed to get application entitlements: %w", err)
	} else if entitlements == nil {
		return []types.ApplicationEntitlement{}, nil
	} else {
		return entitlements, nil
	}
}

func getOverviewArtifacts(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	orgID, customerOrgID := *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()

	artifacts, err := visibleArtifacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}
	artifactIDs := make([]uuid.UUID, len(artifacts))
	for i, a := range artifacts {
		artifactIDs[i] = a.ID
	}
	pulls, err := db.GetLatestArtifactPullsOfCustomerOrganization(ctx, orgID, customerOrgID, artifactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact pulls: %w", err)
	}

	licensing := auth.CurrentOrg().HasFeature(types.FeatureLicensing)
	response := api.OverviewArtifactsResponse{Artifacts: []api.OverviewArtifact{}, HasPulled: len(pulls) > 0}
	for _, artifact := range artifacts {
		if artifact.HiddenInCustomerOverview {
			continue
		}
		versions, err := customerArtifactVersions(ctx, artifact.ID, customerOrgID, licensing)
		if err != nil {
			return nil, fmt.Errorf("failed to get versions for artifact: %w", err)
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
	return respond.JSON(response), nil
}

func getOverviewChecklist(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if checklist, err := db.GetOverviewChecklist(
		ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID(), auth.CurrentUserID(),
	); err != nil {
		return nil, fmt.Errorf("failed to get overview checklist: %w", err)
	} else {
		return respond.JSON(mapping.OverviewChecklistToAPI(*checklist)), nil
	}
}

func getOverviewLicense(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if license, err := db.GetOverviewLicense(ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get license: %w", err)
	} else if license == nil {
		return respond.NoContent(), nil
	} else {
		return respond.JSON(mapping.OverviewLicenseToAPI(*license)), nil
	}
}

func getOverviewAlerts(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if summary, err := db.GetOverviewAlertSummary(ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get alert summary: %w", err)
	} else {
		return respond.JSON(mapping.OverviewAlertSummaryToAPI(*summary)), nil
	}
}

func getOverviewAccessTokens(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if tokens, err := db.GetAccessTokens(ctx, auth.CurrentUserID(), *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get access tokens: %w", err)
	} else {
		now := time.Now()
		threshold := now.Add(overviewAccessTokenExpiryWindow)
		result := []api.OverviewAccessToken{}
		for _, token := range tokens {
			if expiresAt := token.NextExpiresAt(now); expiresAt != nil && expiresAt.Before(threshold) {
				result = append(result, mapping.OverviewAccessTokenToAPI(token, *expiresAt))
			}
		}
		slices.SortFunc(result, func(a, b api.OverviewAccessToken) int { return a.ExpiresAt.Compare(b.ExpiresAt) })
		return respond.JSON(result), nil
	}
}

func getOverviewSupportBundles(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if bundles, err := db.GetOverviewSupportBundles(ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get support bundles: %w", err)
	} else {
		return respond.JSON(mapping.List(bundles, mapping.OverviewSupportBundleToAPI)), nil
	}
}

func getOverviewActivity(r *http.Request) (respond.Responder, error) {
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
		return nil, fmt.Errorf("failed to check for application entitlements: %w", err)
	} else if filter.EntitledArtifactsOnly, err = artifactEntitlementsApply(ctx, filter.Licensing); err != nil {
		return nil, fmt.Errorf("failed to check for artifact entitlements: %w", err)
	} else if activity, err := db.GetOverviewActivity(ctx, org.ID, customerOrg.ID, filter); err != nil {
		return nil, fmt.Errorf("failed to get overview activity: %w", err)
	} else {
		return respond.JSON(mapping.List(activity, mapping.OverviewActivityToAPI)), nil
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

func artifactEntitlementsApply(ctx context.Context, licensing bool) (bool, error) {
	if !licensing {
		return false, nil
	}
	return db.HasAnyArtifactEntitlement(ctx, *auth.Authentication.Require(ctx).CurrentOrgID())
}
