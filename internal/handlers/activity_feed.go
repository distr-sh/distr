package handlers

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
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
	r.With(middleware.RequireOrgAndRole, middleware.UseReadonlyDB).
		Get("/", respond.Handler(getActivityFeed)).
		With(option.Description("List the recent activity of the current customer organization")).
		With(option.Response(http.StatusOK, []api.ActivityFeedEntry{}))
}

type activitySource func(ctx context.Context, customerOrgID uuid.UUID, since time.Time) ([]activityfeed.Entry, error)

func getActivityFeed(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	org := auth.CurrentOrg()
	since := time.Now().Add(-activityfeed.Window)

	customerOrg := cmp.Or(auth.LensCustomerOrg(), auth.CurrentCustomerOrg())
	if customerOrg == nil {
		return respond.BadRequest("the activity feed is only available in a customer organization"), nil
	}

	var entries []activityfeed.Entry
	collect := func(enabled bool, source activitySource) error {
		if !enabled {
			return nil
		}
		result, err := source(ctx, customerOrg.ID, since)
		entries = append(entries, result...)
		return err
	}

	deploymentsEnabled := customerOrg.HasFeature(types.CustomerOrganizationFeatureDeploymentTargets)
	artifactsEnabled := customerOrg.HasFeature(types.CustomerOrganizationFeatureArtifacts)
	if err := collect(deploymentsEnabled, applicationVersionActivity); err != nil {
		return nil, err
	} else if err := collect(deploymentsEnabled, deploymentRevisionActivity); err != nil {
		return nil, err
	} else if err := collect(artifactsEnabled, artifactActivity); err != nil {
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

func applicationVersionActivity(
	ctx context.Context,
	customerOrgID uuid.UUID,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if shouldGetEntitledOnly, err := applicationEntitlementsApply(ctx); err != nil {
		return nil, fmt.Errorf("failed to check for application entitlements: %w", err)
	} else if versions, err := db.GetApplicationVersionsEntitledToCustomer(
		ctx, *auth.CurrentOrgID(), customerOrgID, shouldGetEntitledOnly, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get application versions: %w", err)
	} else {
		return activityfeed.ApplicationVersions(versions), nil
	}
}

func deploymentRevisionActivity(
	ctx context.Context,
	customerOrgID uuid.UUID,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if revisions, err := db.GetCustomerDeploymentRevisions(
		ctx, *auth.CurrentOrgID(), customerOrgID, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get deployment revisions: %w", err)
	} else {
		return activityfeed.DeploymentRevisions(revisions), nil
	}
}

func artifactActivity(ctx context.Context, customerOrgID uuid.UUID, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()
	artifacts, err := visibleArtifacts(ctx, &customerOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}
	artifactIDs := make([]uuid.UUID, len(artifacts))
	for i, artifact := range artifacts {
		artifactIDs[i] = artifact.ID
	}
	if tags, err := db.GetArtifactTagsEntitledToCustomer(
		ctx, artifactIDs, customerOrgID, auth.CurrentOrg().HasFeature(types.FeatureLicensing), since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get artifact tags: %w", err)
	} else if pulls, err := db.GetFirstArtifactVersionPullsOfCustomerOrganization(
		ctx, orgID, customerOrgID, artifactIDs, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get first artifact pulls: %w", err)
	} else {
		return append(activityfeed.ArtifactTags(artifacts, tags), activityfeed.FirstPulls(pulls)...), nil
	}
}

func licenseKeyActivity(ctx context.Context, customerOrgID uuid.UUID, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if keys, err := db.GetLicenseKeysByCustomerOrgID(ctx, customerOrgID, *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get license keys: %w", err)
	} else {
		return activityfeed.LicenseKeys(keys, since), nil
	}
}

func supportBundleActivity(
	ctx context.Context,
	customerOrgID uuid.UUID,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if events, err := db.GetCustomerSupportBundleEvents(
		ctx, *auth.CurrentOrgID(), customerOrgID, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get support bundle events: %w", err)
	} else {
		return activityfeed.SupportBundles(events), nil
	}
}
