package handlers

import (
	"cmp"
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
	r.With(middleware.RequireOrgAndRole, middleware.UseReadonlyDB).
		Get("/", respond.Handler(getActivityFeed)).
		With(option.Description(
			"List the recent activity of the current customer organization, or else of the vendor and every " +
				"customer the vendor or partner organization manages, or of only the given one of those customers",
		)).
		With(option.Request(customerScopeQuery{})).
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

// organizationActivitySources return the activity that belongs to no customer organization.
var organizationActivitySources = []activitySource{
	applicationVersionActivity,
	artifactTagActivity,
	internalDeploymentRevisionActivity,
}

// customerActivitySources return the activity of the customer organizations in the scope.
var customerActivitySources = []activitySource{
	customerDeploymentRevisionActivity,
	firstArtifactPullActivity,
	licenseKeyActivity,
	supportBundleActivity,
}

func getActivityFeed(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	if customerOrgID, resp := customerFilterFromQuery(r); resp != nil {
		return resp, nil
	} else if scope, err := activityFeedScope(ctx, customerOrgID); err != nil {
		return nil, err
	} else if entries, err := collectActivity(ctx, scope, activitySources(customerOrgID)); err != nil {
		return nil, err
	} else {
		entries = activityfeed.RestrictToCustomerOrganizations(entries, scope.customerOrgs)
		entries = activityfeed.Latest(entries)
		return respond.JSON(mapping.List(entries, mapping.ActivityFeedEntryToAPI)), nil
	}
}

// activityFeedScope narrows the scope of the caller down to the customer organization filter if one is given,
// which matches nothing when the caller cannot reach it.
func activityFeedScope(ctx context.Context, filter *uuid.UUID) (activityScope, error) {
	scope, err := callerActivityScope(ctx)
	if err != nil || filter == nil {
		return scope, err
	}
	scope.customerOrgs = slices.DeleteFunc(scope.customerOrgs, func(customerOrg types.CustomerOrganization) bool {
		return customerOrg.ID != *filter
	})
	return scope, nil
}

// activitySources leaves the activity that belongs to no customer organization out of the feed of one customer.
func activitySources(filter *uuid.UUID) []activitySource {
	if filter != nil {
		return customerActivitySources
	}
	return slices.Concat(organizationActivitySources, customerActivitySources)
}

func callerActivityScope(ctx context.Context) (activityScope, error) {
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

func collectActivity(
	ctx context.Context,
	scope activityScope,
	sources []activitySource,
) ([]activityfeed.Entry, error) {
	since := time.Now().Add(-activityfeed.Window)
	var entries []activityfeed.Entry
	for _, source := range sources {
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

func customerDeploymentRevisionActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	customerOrgIDs := scope.customerOrgIDsWith(types.CustomerOrganizationFeatureDeploymentTargets)
	if len(customerOrgIDs) == 0 {
		return nil, nil
	}
	return deploymentRevisionActivity(ctx, scope, customerOrgIDs, false, since)
}

// internalDeploymentRevisionActivity returns the deployment revisions of the deployment targets that belong to no
// customer organization, which only the vendor sees.
func internalDeploymentRevisionActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	if !scope.viewer.IsVendor {
		return nil, nil
	}
	return deploymentRevisionActivity(ctx, scope, nil, true, since)
}

func deploymentRevisionActivity(
	ctx context.Context,
	scope activityScope,
	customerOrgIDs []uuid.UUID,
	shouldIncludeInternal bool,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if revisions, err := db.GetDeploymentRevisionEvents(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, shouldIncludeInternal, scope.viewer, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get deployment revisions: %w", err)
	} else {
		return activityfeed.DeploymentRevisions(revisions), nil
	}
}

// artifactTagActivity returns the new tags of the artifacts, to a customer organization only those it may pull.
func artifactTagActivity(ctx context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if scope.customerOrg != nil && !scope.customerOrg.HasFeature(types.CustomerOrganizationFeatureArtifacts) {
		return nil, nil
	} else if artifacts, err := visibleArtifacts(ctx, scope.customerOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	} else if tags, err := db.GetArtifactTagEvents(
		ctx, artifactIDsOf(artifacts), scope.customerOrgID(), auth.CurrentOrg().HasFeature(types.FeatureLicensing),
		scope.viewer, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get artifact tags: %w", err)
	} else {
		return activityfeed.ArtifactTags(artifacts, tags), nil
	}
}

func firstArtifactPullActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if customerOrgIDs := scope.customerOrgIDsWith(types.CustomerOrganizationFeatureArtifacts); len(customerOrgIDs) == 0 {
		return nil, nil
	} else if artifacts, err := visibleArtifacts(ctx, scope.customerOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	} else if pulls, err := db.GetFirstArtifactVersionPullsOfCustomerOrganizations(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, artifactIDsOf(artifacts), since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get first artifact pulls: %w", err)
	} else {
		return activityfeed.FirstPulls(pulls), nil
	}
}

func artifactIDsOf(artifacts []types.ArtifactWithDownloads) []uuid.UUID {
	ids := make([]uuid.UUID, len(artifacts))
	for i, artifact := range artifacts {
		ids[i] = artifact.ID
	}
	return ids
}

func licenseKeyActivity(ctx context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	customerOrgIDs := make([]uuid.UUID, len(scope.customerOrgs))
	for i, customerOrg := range scope.customerOrgs {
		customerOrgIDs[i] = customerOrg.ID
	}
	if !auth.CurrentOrg().HasFeature(types.FeatureLicensing) || len(customerOrgIDs) == 0 {
		return nil, nil
	} else if keys, err := db.GetLicenseKeyEvents(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get license keys: %w", err)
	} else {
		return activityfeed.LicenseKeys(keys), nil
	}
}

func supportBundleActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	customerOrgIDs := scope.customerOrgIDsWith(types.CustomerOrganizationFeatureSupportBundles)
	if !auth.CurrentOrg().HasFeature(types.FeatureSupport) || len(customerOrgIDs) == 0 {
		return nil, nil
	} else if events, err := db.GetCustomerSupportBundleEvents(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, scope.viewer, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get support bundle events: %w", err)
	} else {
		return activityfeed.SupportBundles(events), nil
	}
}
