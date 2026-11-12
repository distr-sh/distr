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
			"List the recent activity of the current customer organization, or else of the vendor and every " +
				"customer the vendor or partner organization manages",
		)).
		With(option.Response(http.StatusOK, []api.ActivityFeedEntry{}))
}

// activityScope is what an activity feed covers: the customer organization a request looks at, or the vendor
// and every customer organization of the vendor or partner.
type activityScope struct {
	// customerOrg is the customer organization the request looks at, nil for a vendor or partner.
	customerOrg  *types.CustomerOrganization
	customerOrgs []types.CustomerOrganization
	viewer       types.ActivityViewer
}

func (s activityScope) customerOrgID() *uuid.UUID {
	if s.customerOrg == nil {
		return nil
	}
	return &s.customerOrg.ID
}

func (s activityScope) customerOrgIDsWith(feature types.CustomerOrganizationFeature) []uuid.UUID {
	var result []uuid.UUID
	for _, customerOrg := range s.customerOrgs {
		if customerOrg.HasFeature(feature) {
			result = append(result, customerOrg.ID)
		}
	}
	return result
}

type activitySource func(ctx context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error)

func getActivityFeed(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	if scope, err := activityFeedScope(ctx); err != nil {
		return nil, err
	} else if entries, err := collectActivity(ctx, scope); err != nil {
		return nil, err
	} else {
		latest := activityfeed.Latest(entries, scope.customerOrgs)
		return respond.JSON(mapping.List(latest, mapping.ActivityFeedEntryToAPI)), nil
	}
}

func activityFeedScope(ctx context.Context) (activityScope, error) {
	auth := auth.Authentication.Require(ctx)
	if customerOrg := cmp.Or(auth.LensCustomerOrg(), auth.CurrentCustomerOrg()); customerOrg != nil {
		return activityScope{customerOrg: customerOrg, customerOrgs: []types.CustomerOrganization{*customerOrg}}, nil
	}

	partnerOrgID := auth.CurrentPartnerOrgID()
	var withUsage []types.CustomerOrganizationWithUsage
	var err error
	if partnerOrgID != nil {
		withUsage, err = db.GetCustomerOrganizationsByPartnerOrgID(ctx, *partnerOrgID)
	} else {
		withUsage, err = db.GetCustomerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID())
	}
	if err != nil {
		return activityScope{}, fmt.Errorf("failed to get customer organizations: %w", err)
	}
	scope := activityScope{
		customerOrgs: make([]types.CustomerOrganization, len(withUsage)),
		viewer:       types.ActivityViewer{IsVendor: partnerOrgID == nil, PartnerOrganizationID: partnerOrgID},
	}
	for i, customerOrg := range withUsage {
		scope.customerOrgs[i] = customerOrg.CustomerOrganization
	}
	return scope, nil
}

func collectActivity(ctx context.Context, scope activityScope) ([]activityfeed.Entry, error) {
	since := time.Now().Add(-activityfeed.Window)
	var entries []activityfeed.Entry
	for _, source := range []activitySource{
		applicationVersionActivity,
		deploymentRevisionActivity,
		artifactActivity,
		licenseKeyActivity,
		supportBundleActivity,
	} {
		if result, err := source(ctx, scope, since); err != nil {
			return nil, err
		} else {
			entries = append(entries, result...)
		}
	}
	return entries, nil
}

// applicationVersionActivity returns the new versions of the vendor's applications, to a customer organization
// only those it may deploy.
func applicationVersionActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if scope.customerOrg != nil && !scope.customerOrg.HasFeature(types.CustomerOrganizationFeatureDeploymentTargets) {
		return nil, nil
	} else if shouldGetEntitledOnly, err := applicationEntitlementsApply(ctx); err != nil {
		return nil, fmt.Errorf("failed to check for application entitlements: %w", err)
	} else if versions, err := db.GetRecentApplicationVersions(
		ctx, *auth.CurrentOrgID(), scope.customerOrgID(), shouldGetEntitledOnly, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get application versions: %w", err)
	} else {
		return activityfeed.ApplicationVersions(versions), nil
	}
}

func deploymentRevisionActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	customerOrgIDs := scope.customerOrgIDsWith(types.CustomerOrganizationFeatureDeploymentTargets)
	if len(customerOrgIDs) == 0 && !scope.viewer.IsVendor {
		return nil, nil
	} else if revisions, err := db.GetDeploymentRevisionEvents(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, scope.viewer, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get deployment revisions: %w", err)
	} else {
		return activityfeed.DeploymentRevisions(revisions), nil
	}
}

// artifactActivity returns the new tags of the artifacts, to a customer organization only those it may pull, and
// the first pulls of the customer organizations.
func artifactActivity(ctx context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if scope.customerOrg != nil && !scope.customerOrg.HasFeature(types.CustomerOrganizationFeatureArtifacts) {
		return nil, nil
	}
	artifacts, err := visibleArtifacts(ctx, scope.customerOrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	}
	artifactIDs := make([]uuid.UUID, len(artifacts))
	for i, artifact := range artifacts {
		artifactIDs[i] = artifact.ID
	}
	tags, err := db.GetRecentArtifactTags(
		ctx, artifactIDs, scope.customerOrgID(), auth.CurrentOrg().HasFeature(types.FeatureLicensing),
		since, activityfeed.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact tags: %w", err)
	}
	entries := activityfeed.ArtifactTags(artifacts, tags)
	if customerOrgIDs := scope.customerOrgIDsWith(types.CustomerOrganizationFeatureArtifacts); len(customerOrgIDs) > 0 {
		if pulls, err := db.GetFirstArtifactVersionPullsOfCustomerOrganizations(
			ctx, *auth.CurrentOrgID(), customerOrgIDs, artifactIDs, since, activityfeed.Limit,
		); err != nil {
			return nil, fmt.Errorf("failed to get first artifact pulls: %w", err)
		} else {
			entries = append(entries, activityfeed.FirstPulls(pulls)...)
		}
	}
	return entries, nil
}

func licenseKeyActivity(ctx context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	customerOrgIDs := make([]uuid.UUID, len(scope.customerOrgs))
	for i, customerOrg := range scope.customerOrgs {
		customerOrgIDs[i] = customerOrg.ID
	}
	if !auth.CurrentOrg().HasFeature(types.FeatureLicensing) || len(customerOrgIDs) == 0 {
		return nil, nil
	} else if keys, err := db.GetLicenseKeysByCustomerOrgIDs(ctx, customerOrgIDs, *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get license keys: %w", err)
	} else {
		return activityfeed.LicenseKeys(keys, since), nil
	}
}

func supportBundleActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	customerOrgIDs := scope.customerOrgIDsWith(types.CustomerOrganizationFeatureSupportBundles)
	if !auth.CurrentOrg().SubscriptionType.IsPro() || len(customerOrgIDs) == 0 {
		return nil, nil
	} else if events, err := db.GetCustomerSupportBundleEvents(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, scope.viewer, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get support bundle events: %w", err)
	} else {
		return activityfeed.SupportBundles(events), nil
	}
}
