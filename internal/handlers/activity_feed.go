package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/activityfeed"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
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
		With(option.Description("List the recent activity of a customer organization")).
		With(option.Request(customerScopeQuery{})).
		With(option.Response(http.StatusOK, []api.ActivityFeedEntry{}))
}

type activitySource func(ctx context.Context, customerOrgID uuid.UUID, since time.Time) ([]activityfeed.Entry, error)

func getActivityFeed(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	org := auth.Authentication.Require(ctx).CurrentOrg()
	since := time.Now().Add(-activityfeed.Window)

	customerOrg, resp, err := activityFeedCustomer(r)
	if resp != nil || err != nil {
		return resp, err
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

// activityFeedCustomer returns the customer organization of a customer user, or the one a vendor or partner
// user asks for.
func activityFeedCustomer(r *http.Request) (*types.CustomerOrganization, respond.Responder, error) {
	ctx := r.Context()
	if customerOrgID, resp, err := customerScopeFromQuery(r); resp != nil || err != nil {
		return nil, resp, err
	} else if customerOrgID == nil {
		return nil, respond.BadRequest("customer organization ID is required"), nil
	} else if customerOrg := auth.Authentication.Require(ctx).CurrentCustomerOrg(); customerOrg != nil {
		return customerOrg, nil, nil
	} else if customerOrg, err := db.GetCustomerOrganizationByID(ctx, *customerOrgID); errors.Is(
		err, distrerr.ErrNotFound,
	) {
		return nil, invalidCustomerScope, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get customer organization: %w", err)
	} else {
		return &customerOrg.CustomerOrganization, nil, nil
	}
}

func applicationVersionActivity(
	ctx context.Context,
	customerOrgID uuid.UUID,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if entitledOnly, err := applicationEntitlementsApply(ctx); err != nil {
		return nil, fmt.Errorf("failed to check for application entitlements: %w", err)
	} else if versions, err := db.GetCustomerApplicationVersions(
		ctx, *auth.CurrentOrgID(), customerOrgID, entitledOnly, since, activityfeed.Limit,
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
	releaseVersions, err := db.ListArtifactReleaseVersions(
		ctx, artifactIDs, customerOrgID, auth.CurrentOrg().HasFeature(types.FeatureLicensing),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact versions: %w", err)
	}
	pulls, err := db.GetFirstArtifactVersionPullsOfCustomerOrganization(
		ctx, orgID, customerOrgID, artifactIDs, since, activityfeed.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get first artifact pulls: %w", err)
	}

	versions := types.TaggedArtifactVersionsByArtifact(releaseVersions)
	var result []activityfeed.Entry
	for _, artifact := range artifacts {
		result = append(result, activityfeed.ArtifactVersions(artifact, versions[artifact.ID], since)...)
	}
	return append(result, activityfeed.FirstPulls(pulls)...), nil
}

func licenseKeyActivity(ctx context.Context, customerOrgID uuid.UUID, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if revisions, err := db.GetCustomerLicenseKeyRevisions(
		ctx, *auth.CurrentOrgID(), customerOrgID, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get license key revisions: %w", err)
	} else {
		return activityfeed.LicenseKeyRevisions(revisions), nil
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
