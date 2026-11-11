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
		With(option.Description(
			"List the recent activity of the current customer organization, or else of every customer the " +
				"vendor or partner organization manages",
		)).
		With(option.Response(http.StatusOK, []api.ActivityFeedEntry{}))
}

type activitySource func(ctx context.Context, customerOrgIDs []uuid.UUID, since time.Time) ([]activityfeed.Entry, error)

func getActivityFeed(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	if customerOrgs, err := activityFeedCustomerOrgs(ctx); err != nil {
		return nil, err
	} else if entries, err := collectActivity(ctx, customerOrgs); err != nil {
		return nil, err
	} else {
		return respond.JSON(mapping.List(activityfeed.Latest(entries, customerOrgs), mapping.ActivityFeedEntryToAPI)), nil
	}
}

// collectActivity asks every source for the activity of the customer organizations that the source is enabled
// for, at once.
func collectActivity(ctx context.Context, customerOrgs []types.CustomerOrganization) ([]activityfeed.Entry, error) {
	org := auth.Authentication.Require(ctx).CurrentOrg()
	since := time.Now().Add(-activityfeed.Window)
	withFeature := func(feature types.CustomerOrganizationFeature) func(types.CustomerOrganization) bool {
		return func(customerOrg types.CustomerOrganization) bool { return customerOrg.HasFeature(feature) }
	}
	sources := []struct {
		source  activitySource
		enabled func(types.CustomerOrganization) bool
	}{
		{applicationVersionActivity, withFeature(types.CustomerOrganizationFeatureDeploymentTargets)},
		{deploymentRevisionActivity, withFeature(types.CustomerOrganizationFeatureDeploymentTargets)},
		{artifactActivity, withFeature(types.CustomerOrganizationFeatureArtifacts)},
		{licenseKeyActivity, func(types.CustomerOrganization) bool { return org.HasFeature(types.FeatureLicensing) }},
		{supportBundleActivity, func(customerOrg types.CustomerOrganization) bool {
			return org.SubscriptionType.IsPro() && customerOrg.HasFeature(types.CustomerOrganizationFeatureSupportBundles)
		}},
	}

	var entries []activityfeed.Entry
	for _, s := range sources {
		var customerOrgIDs []uuid.UUID
		for _, customerOrg := range customerOrgs {
			if s.enabled(customerOrg) {
				customerOrgIDs = append(customerOrgIDs, customerOrg.ID)
			}
		}
		if len(customerOrgIDs) == 0 {
			continue
		}
		if result, err := s.source(ctx, customerOrgIDs, since); err != nil {
			return nil, err
		} else {
			entries = append(entries, result...)
		}
	}
	return entries, nil
}

// activityFeedCustomerOrgs returns the customer organization the request looks at, or else every customer
// organization of the partner or vendor organization.
func activityFeedCustomerOrgs(ctx context.Context) ([]types.CustomerOrganization, error) {
	auth := auth.Authentication.Require(ctx)
	if customerOrg := cmp.Or(auth.LensCustomerOrg(), auth.CurrentCustomerOrg()); customerOrg != nil {
		return []types.CustomerOrganization{*customerOrg}, nil
	}

	var withUsage []types.CustomerOrganizationWithUsage
	var err error
	if partnerOrgID := auth.CurrentPartnerOrgID(); partnerOrgID != nil {
		withUsage, err = db.GetCustomerOrganizationsByPartnerOrgID(ctx, *partnerOrgID)
	} else {
		withUsage, err = db.GetCustomerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get customer organizations: %w", err)
	}
	result := make([]types.CustomerOrganization, len(withUsage))
	for i, customerOrg := range withUsage {
		result[i] = customerOrg.CustomerOrganization
	}
	return result, nil
}

func applicationVersionActivity(
	ctx context.Context,
	customerOrgIDs []uuid.UUID,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if shouldGetEntitledOnly, err := applicationEntitlementsApply(ctx); err != nil {
		return nil, fmt.Errorf("failed to check for application entitlements: %w", err)
	} else if versions, err := db.GetApplicationVersionsEntitledToCustomers(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, shouldGetEntitledOnly, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get application versions: %w", err)
	} else {
		return activityfeed.ApplicationVersions(versions), nil
	}
}

func deploymentRevisionActivity(
	ctx context.Context,
	customerOrgIDs []uuid.UUID,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if revisions, err := db.GetCustomerDeploymentRevisions(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get deployment revisions: %w", err)
	} else {
		return activityfeed.DeploymentRevisions(revisions), nil
	}
}

func artifactActivity(
	ctx context.Context,
	customerOrgIDs []uuid.UUID,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	orgID := *auth.CurrentOrgID()
	licensingEnabled := auth.CurrentOrg().HasFeature(types.FeatureLicensing)
	if tags, err := db.GetArtifactTagsEntitledToCustomers(
		ctx, orgID, customerOrgIDs, licensingEnabled, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get artifact tags: %w", err)
	} else if entitlementsApply, err := artifactEntitlementsApply(ctx); err != nil {
		return nil, fmt.Errorf("failed to check for artifact entitlements: %w", err)
	} else if pulls, err := db.GetFirstArtifactVersionPullsOfCustomerOrganizations(
		ctx, orgID, customerOrgIDs, entitlementsApply, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get first artifact pulls: %w", err)
	} else {
		return append(activityfeed.ArtifactTags(tags), activityfeed.FirstPulls(pulls)...), nil
	}
}

func licenseKeyActivity(
	ctx context.Context,
	customerOrgIDs []uuid.UUID,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if keys, err := db.GetLicenseKeysByCustomerOrgIDs(ctx, customerOrgIDs, *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get license keys: %w", err)
	} else {
		return activityfeed.LicenseKeys(keys, since), nil
	}
}

func supportBundleActivity(
	ctx context.Context,
	customerOrgIDs []uuid.UUID,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if events, err := db.GetCustomerSupportBundleEvents(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get support bundle events: %w", err)
	} else {
		return activityfeed.SupportBundles(events), nil
	}
}
