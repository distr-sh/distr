package handlers

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/activityfeed"
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

func ActivityFeedRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.With(middleware.RequireOrgAndRole, middleware.RequireCustomer, middleware.UseReadonlyDB).
		Get("/", respond.Handler(getActivityFeed)).
		With(option.Description("List the recent activity of the customer organization")).
		With(option.Response(http.StatusOK, []api.ActivityFeedEntry{}))
}

func getActivityFeed(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	org, customerOrg := auth.CurrentOrg(), auth.CurrentCustomerOrg()
	since := time.Now().Add(-activityfeed.Window)

	var entries []activityfeed.Entry
	collect := func(enabled bool, source func(context.Context, time.Time) ([]activityfeed.Entry, error)) error {
		if !enabled {
			return nil
		}
		result, err := source(ctx, since)
		entries = append(entries, result...)
		return err
	}

	deployments := customerOrg.HasFeature(types.CustomerOrganizationFeatureDeploymentTargets)
	artifacts := customerOrg.HasFeature(types.CustomerOrganizationFeatureArtifacts)
	if err := collect(deployments, applicationVersionActivity); err != nil {
		return nil, err
	} else if err := collect(deployments, deploymentRevisionActivity); err != nil {
		return nil, err
	} else if err := collect(artifacts, artifactActivity); err != nil {
		return nil, err
	} else if err := collect(org.HasFeature(types.FeatureLicensing), licenseKeyActivity); err != nil {
		return nil, err
	} else if err := collect(
		org.SubscriptionType.IsPro() && customerOrg.HasFeature(types.CustomerOrganizationFeatureSupportBundles),
		supportBundleActivity,
	); err != nil {
		return nil, err
	} else {
		return respond.JSON(mapping.List(activityfeed.Latest(entries), mapping.ActivityFeedEntryToAPI)), nil
	}
}

func applicationVersionActivity(ctx context.Context, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if entitledOnly, err := applicationEntitlementsApply(ctx); err != nil {
		return nil, fmt.Errorf("failed to check for application entitlements: %w", err)
	} else if versions, err := db.GetCustomerApplicationVersions(
		ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID(), entitledOnly, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get application versions: %w", err)
	} else {
		return activityfeed.ApplicationVersions(versions), nil
	}
}

func deploymentRevisionActivity(ctx context.Context, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if revisions, err := db.GetCustomerDeploymentRevisions(
		ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID(), since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get deployment revisions: %w", err)
	} else {
		return activityfeed.DeploymentRevisions(revisions), nil
	}
}

func artifactActivity(ctx context.Context, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	orgID, customerOrgID := *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID()
	artifacts, err := visibleArtifacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}
	artifactIDs := make([]uuid.UUID, len(artifacts))
	for i, artifact := range artifacts {
		artifactIDs[i] = artifact.ID
	}
	releaseVersions, err := db.ListArtifactReleaseVersions(
		ctx, artifactIDs, customerOrgID, auth.CurrentOrg().HasFeature(types.FeatureLicensing),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact versions: %w", err)
	}
	pulls, err := db.GetFirstArtifactVersionPullsOfCustomerOrganization(
		ctx, orgID, customerOrgID, since, activityfeed.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get first artifact pulls: %w", err)
	}

	versions := types.TaggedArtifactVersionsByArtifact(releaseVersions)
	var result []activityfeed.Entry
	for _, artifact := range artifacts {
		result = append(result, activityfeed.ArtifactVersions(artifact, versions[artifact.ID], since)...)
	}
	for _, entry := range activityfeed.FirstPulls(pulls) {
		if slices.Contains(artifactIDs, entry.ResourceID) {
			result = append(result, entry)
		}
	}
	return result, nil
}

func licenseKeyActivity(ctx context.Context, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if revisions, err := db.GetCustomerLicenseKeyRevisions(
		ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID(), since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get license key revisions: %w", err)
	} else {
		return activityfeed.LicenseKeyRevisions(revisions), nil
	}
}

func supportBundleActivity(ctx context.Context, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if events, err := db.GetCustomerSupportBundleEvents(
		ctx, *auth.CurrentOrgID(), *auth.CurrentCustomerOrgID(), since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get support bundle events: %w", err)
	} else {
		return activityfeed.SupportBundles(events), nil
	}
}
