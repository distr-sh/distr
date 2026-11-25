package handlers

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
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
	"github.com/distr-sh/distr/internal/util"
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
	customerOrg *types.CustomerOrganization
	// partnerOrgID is the partner organization of a partner, nil for the vendor and a customer organization.
	partnerOrgID *uuid.UUID
	customerOrgs []types.CustomerOrganization
	// artifacts loads the artifacts visible to the scope once for every source that needs them.
	artifacts func() ([]types.ArtifactWithDownloads, error)
	// users loads the members of the scope once for every source that needs them: those of the single customer
	// organization a feed covers, or else every user the vendor or partner manages, its own members included.
	users func() ([]types.UserAccountWithUserRole, error)
}

func (s activityScope) customerOrgID() *uuid.UUID {
	if s.customerOrg == nil {
		return nil
	}
	return &s.customerOrg.ID
}

func (s activityScope) isVendor() bool {
	return s.customerOrg == nil && s.partnerOrgID == nil
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
	internalDeploymentTargetActivity,
	teamMemberActivity,
}

// customerActivitySources return the activity of the customer organizations in the scope.
var customerActivitySources = []activitySource{
	customerCreatedActivity,
	customerUserActivity,
	customerDeploymentRevisionActivity,
	customerDeploymentTargetActivity,
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
	} else if entries, err := collectActivity(ctx, scope, activitySources(customerOrgID != nil)); err != nil {
		return nil, err
	} else {
		entries = activityfeed.RestrictToCustomerOrganizations(entries, scope.customerOrgs)
		entries = activityfeed.Latest(entries)
		return respond.JSON(mapping.List(
			entries, mapping.ActivityFeedEntryToAPI(scope.customerOrgID(), scope.partnerOrgID),
		)), nil
	}
}

// activityFeedScope narrows the scope of the caller down to the given customer organization if there is one,
// which matches nothing when the caller cannot reach it.
func activityFeedScope(ctx context.Context, customerOrgID *uuid.UUID) (activityScope, error) {
	scope, err := callerActivityScope(ctx, customerOrgID)
	if err != nil {
		return scope, err
	}
	ownCustomerOrgID := scope.customerOrgID()
	scope.artifacts = sync.OnceValues(func() ([]types.ArtifactWithDownloads, error) {
		return visibleArtifacts(ctx, ownCustomerOrgID)
	})
	isSingleCustomer := customerOrgID != nil || scope.customerOrg != nil
	customerOrgs := scope.customerOrgs
	partnerOrgID := scope.partnerOrgID
	scope.users = sync.OnceValues(func() ([]types.UserAccountWithUserRole, error) {
		return activityUsers(ctx, isSingleCustomer, customerOrgs, partnerOrgID)
	})
	return scope, nil
}

func activityUsers(
	ctx context.Context,
	isSingleCustomer bool,
	customerOrgs []types.CustomerOrganization,
	partnerOrgID *uuid.UUID,
) ([]types.UserAccountWithUserRole, error) {
	switch {
	case isSingleCustomer && len(customerOrgs) == 0:
		return nil, nil
	case isSingleCustomer:
		return db.GetUserAccountsByCustomerOrgID(ctx, customerOrgs[0].ID)
	case partnerOrgID != nil:
		return db.GetUserAccountsByPartnerOrgID(ctx, *partnerOrgID)
	default:
		return db.GetUserAccountsByOrgID(ctx, *auth.Authentication.Require(ctx).CurrentOrgID())
	}
}

// activitySources leaves the activity that belongs to no customer organization out of the feed of one customer.
func activitySources(isSingleCustomer bool) []activitySource {
	if isSingleCustomer {
		return customerActivitySources
	}
	return slices.Concat(organizationActivitySources, customerActivitySources)
}

func callerActivityScope(ctx context.Context, customerOrgID *uuid.UUID) (activityScope, error) {
	auth := auth.Authentication.Require(ctx)
	if customerOrg := cmp.Or(auth.LensCustomerOrg(), auth.CurrentCustomerOrg()); customerOrg != nil {
		scope := activityScope{customerOrg: customerOrg}
		if customerOrgID == nil || *customerOrgID == customerOrg.ID {
			scope.customerOrgs = []types.CustomerOrganization{*customerOrg}
		}
		return scope, nil
	}

	partnerOrgID := auth.CurrentPartnerOrgID()
	scope := activityScope{partnerOrgID: partnerOrgID}
	var err error
	if customerOrgID != nil {
		scope.customerOrgs, err = managedCustomerOrg(ctx, *customerOrgID, *auth.CurrentOrgID(), partnerOrgID)
	} else {
		scope.customerOrgs, err = managedCustomerOrgs(ctx, *auth.CurrentOrgID(), partnerOrgID)
	}
	return scope, err
}

// managedCustomerOrg returns the customer organization if the vendor or partner manages it, or else none.
func managedCustomerOrg(
	ctx context.Context,
	customerOrgID, orgID uuid.UUID,
	partnerOrgID *uuid.UUID,
) ([]types.CustomerOrganization, error) {
	if customerOrg, err := db.GetCustomerOrganization(ctx, customerOrgID, orgID); errors.Is(err, distrerr.ErrNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get customer organization: %w", err)
	} else if partnerOrgID != nil && !util.PtrEq(customerOrg.PartnerOrganizationID, partnerOrgID) {
		return nil, nil
	} else {
		return []types.CustomerOrganization{*customerOrg}, nil
	}
}

func managedCustomerOrgs(
	ctx context.Context,
	orgID uuid.UUID,
	partnerOrgID *uuid.UUID,
) ([]types.CustomerOrganization, error) {
	var withUsage []types.CustomerOrganizationWithUsage
	var err error
	if partnerOrgID != nil {
		withUsage, err = db.GetCustomerOrganizationsByPartnerOrgID(ctx, *partnerOrgID)
	} else {
		withUsage, err = db.GetCustomerOrganizationsByOrganizationID(ctx, orgID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get customer organizations: %w", err)
	}
	customerOrgs := make([]types.CustomerOrganization, len(withUsage))
	for i, customerOrg := range withUsage {
		customerOrgs[i] = customerOrg.CustomerOrganization
	}
	return customerOrgs, nil
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
	auth := auth.Authentication.Require(ctx)
	if customerOrgIDs := scope.customerOrgIDsWith(
		types.CustomerOrganizationFeatureDeploymentTargets,
	); len(customerOrgIDs) == 0 {
		return nil, nil
	} else if revisions, err := db.GetCustomerDeploymentRevisionEvents(
		ctx, *auth.CurrentOrgID(), customerOrgIDs, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get customer deployment revisions: %w", err)
	} else {
		return activityfeed.DeploymentRevisions(revisions), nil
	}
}

// internalDeploymentRevisionActivity returns the deployment revisions of the deployment targets that belong to no
// customer organization, which only the vendor sees.
func internalDeploymentRevisionActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if !scope.isVendor() {
		return nil, nil
	} else if revisions, err := db.GetInternalDeploymentRevisionEvents(
		ctx, *auth.CurrentOrgID(), since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get internal deployment revisions: %w", err)
	} else {
		return activityfeed.DeploymentRevisions(revisions), nil
	}
}

// customerCreatedActivity returns the new customer organizations of the vendor or partner.
func customerCreatedActivity(_ context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error) {
	if scope.customerOrg != nil {
		return nil, nil
	}
	return activityfeed.CustomerOrganizations(scope.customerOrgs, since), nil
}

func customerUserActivity(_ context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error) {
	if len(scope.customerOrgs) == 0 {
		return nil, nil
	} else if users, err := scope.users(); err != nil {
		return nil, fmt.Errorf("failed to get customer users: %w", err)
	} else {
		return activityfeed.CustomerUsers(users, since), nil
	}
}

// teamMemberActivity returns the new members of the vendor and its partners to the vendor, and those of its own
// organization to a partner.
func teamMemberActivity(ctx context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error) {
	if scope.customerOrg != nil {
		return nil, nil
	} else if users, err := scope.users(); err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	} else if partnerOrgs, err := teamPartnerOrgs(ctx, scope, users); err != nil {
		return nil, err
	} else {
		return activityfeed.TeamMembers(users, partnerOrgs, since), nil
	}
}

// teamPartnerOrgs returns the partner organizations the vendor sees the new members of named with, which a
// partner does not need for its own members.
func teamPartnerOrgs(
	ctx context.Context,
	scope activityScope,
	users []types.UserAccountWithUserRole,
) ([]types.PartnerOrganizationWithUsage, error) {
	auth := auth.Authentication.Require(ctx)
	if !scope.isVendor() || !slices.ContainsFunc(users, func(user types.UserAccountWithUserRole) bool {
		return user.PartnerOrganizationID != nil
	}) {
		return nil, nil
	} else if partnerOrgs, err := db.GetPartnerOrganizationsByOrganizationID(ctx, *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get partner organizations: %w", err)
	} else {
		return partnerOrgs, nil
	}
}

func customerDeploymentTargetActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if customerOrgIDs := scope.customerOrgIDsWith(
		types.CustomerOrganizationFeatureDeploymentTargets,
	); len(customerOrgIDs) == 0 {
		return nil, nil
	} else if targets, err := db.GetDeploymentTargetsCreatedSince(
		ctx, *auth.CurrentOrgID(), db.DeploymentTargetFilter{CustomerOrganizationIDs: customerOrgIDs},
		since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get customer deployment targets: %w", err)
	} else {
		return activityfeed.DeploymentTargets(targets), nil
	}
}

// internalDeploymentTargetActivity returns the new deployment targets that belong to no customer organization,
// which only the vendor sees.
func internalDeploymentTargetActivity(
	ctx context.Context,
	scope activityScope,
	since time.Time,
) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if !scope.isVendor() {
		return nil, nil
	} else if targets, err := db.GetDeploymentTargetsCreatedSince(
		ctx, *auth.CurrentOrgID(), db.DeploymentTargetFilter{Internal: true}, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get internal deployment targets: %w", err)
	} else {
		return activityfeed.DeploymentTargets(targets), nil
	}
}

// artifactTagActivity returns the new tags of the artifacts, to a customer organization only those it may pull.
func artifactTagActivity(ctx context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if scope.customerOrg != nil && !scope.customerOrg.HasFeature(types.CustomerOrganizationFeatureArtifacts) {
		return nil, nil
	} else if artifacts, err := scope.artifacts(); err != nil {
		return nil, fmt.Errorf("failed to get artifacts: %w", err)
	} else if tags, err := db.GetArtifactTagEvents(
		ctx, artifactIDsOf(artifacts), scope.customerOrgID(), auth.CurrentOrg().HasFeature(types.FeatureLicensing),
		since, activityfeed.Limit,
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
	} else if artifacts, err := scope.artifacts(); err != nil {
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

func customerOrgIDsOf(customerOrgs []types.CustomerOrganization) []uuid.UUID {
	ids := make([]uuid.UUID, len(customerOrgs))
	for i, customerOrg := range customerOrgs {
		ids[i] = customerOrg.ID
	}
	return ids
}

func licenseKeyActivity(ctx context.Context, scope activityScope, since time.Time) ([]activityfeed.Entry, error) {
	auth := auth.Authentication.Require(ctx)
	if !auth.CurrentOrg().HasFeature(types.FeatureLicensing) || len(scope.customerOrgs) == 0 {
		return nil, nil
	} else if keys, err := db.GetLicenseKeyEvents(
		ctx, *auth.CurrentOrgID(), customerOrgIDsOf(scope.customerOrgs), since, activityfeed.Limit,
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
		ctx, *auth.CurrentOrgID(), customerOrgIDs, since, activityfeed.Limit,
	); err != nil {
		return nil, fmt.Errorf("failed to get support bundle events: %w", err)
	} else {
		return activityfeed.SupportBundles(events), nil
	}
}
