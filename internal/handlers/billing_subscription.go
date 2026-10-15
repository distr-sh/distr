package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/billing"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v86"
	"go.uber.org/zap"
)

func GetSubscriptionHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	org := auth.CurrentOrg()

	if info, err := buildSubscriptionInfo(ctx, org); err != nil {
		return nil, fmt.Errorf("failed to build subscription info: %w", err)
	} else {
		return respond.JSON(info), nil
	}
}

func CreateSubscriptionHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if body, resp := decodeJSON[api.CreateSubscriptionRequest](r); resp != nil {
		return resp, nil
	} else if usage, err := getCurrentUsageCounts(ctx, *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("failed to get current usage counts: %w", err)
	} else if err := validateSubscriptionQuantities(
		body.SubscriptionType,
		body.CustomerOrganizationQty,
		body.UserAccountQty,
		usage,
	); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if session, err := billing.CreateCheckoutSession(ctx, billing.CheckoutSessionParams{
		OrganizationID:          auth.CurrentOrgID().String(),
		TrialEndsAt:             auth.CurrentOrg().SubscriptionEndsAt,
		SubscriptionType:        body.SubscriptionType,
		SubscriptionPeriod:      body.SubscriptionPeriod,
		CustomerOrganizationQty: body.CustomerOrganizationQty,
		UserAccountQty:          body.UserAccountQty,
		Currency:                "usd",
		SuccessURL:              fmt.Sprintf("%v/subscription/callback", handlerutil.GetRequestSchemeAndHost(r)),
	}); err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %w", err)
	} else {
		return respond.JSON(api.CheckoutResponse{
			SessionID: session.ID,
			URL:       session.URL,
		}), nil
	}
}

func UpdateSubscriptionHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	org := auth.CurrentOrg()

	if org.SubscriptionType == types.SubscriptionTypeTrial {
		return respond.BadRequest("cannot update trial subscription, please create a new subscription"), nil
	} else if org.StripeSubscriptionID == nil || *org.StripeSubscriptionID == "" {
		return respond.BadRequest("no active subscription found"), nil
	}

	body, resp := decodeJSON[api.UpdateSubscriptionRequest](r)
	if resp != nil {
		return resp, nil
	}

	// Requesting the current type is a no-op quantity update
	if body.SubscriptionType != nil && *body.SubscriptionType == org.SubscriptionType {
		body.SubscriptionType = nil
	}

	var info *api.SubscriptionInfo
	if body.SubscriptionType != nil &&
		(org.SubscriptionType != types.SubscriptionTypePro ||
			*body.SubscriptionType != types.SubscriptionTypeBusiness) {
		// Plan switching is currently limited to the pro → business upgrade
		return respond.BadRequest("only upgrading from the pro to the business plan is supported"), nil
	} else if resp, err := runTxRR(ctx, func(ctx context.Context) (respond.Responder, error) {
		org, err := db.GetOrganizationByID(ctx, org.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get organization: %w", err)
		}

		usage, err := getCurrentUsageCounts(ctx, org.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get current usage counts: %w", err)
		}

		// Quantities are validated against the target plan when switching plans
		targetSubscriptionType := org.SubscriptionType
		if body.SubscriptionType != nil {
			targetSubscriptionType = *body.SubscriptionType
		}

		if err := validateSubscriptionQuantities(
			targetSubscriptionType,
			body.CustomerOrganizationQty,
			body.UserAccountQty,
			usage,
		); err != nil {
			return respond.BadRequest(err.Error()), nil
		}

		updatedSub, err := billing.UpdateSubscription(ctx, billing.SubscriptionUpdateParams{
			SubscriptionID:          *org.StripeSubscriptionID,
			CustomerOrganizationQty: body.CustomerOrganizationQty,
			UserAccountQty:          body.UserAccountQty,
			SubscriptionType:        body.SubscriptionType,
			ReturnURL:               fmt.Sprintf("%v/subscription", handlerutil.GetRequestSchemeAndHost(r)),
		})
		if stripeErr, ok := errors.AsType[*stripe.Error](err); ok && stripeErr.Type == stripe.ErrorTypeInvalidRequest {
			internalctx.GetLogger(ctx).Warn("stripe rejected subscription update", zap.Error(err))
			if stripeErr.Code == stripe.ErrorCodeResourceMissing {
				return respond.BadRequest("stripe error: subscription not found"), nil
			} else {
				return respond.BadRequest("stripe error: invalid request (has the subscription been canceled?)"), nil
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to update subscription: %w", err)
		}

		customerOrgQty, err := billing.GetCustomerOrganizationQty(*updatedSub)
		if err != nil {
			return nil, fmt.Errorf("failed to get customer quantity from updated subscription: %w", err)
		}

		userAccountQty, err := billing.GetUserAccountQty(*updatedSub)
		if err != nil {
			return nil, fmt.Errorf("failed to get user account quantity from updated subscription: %w", err)
		}

		// The subscription type and plan-managed features are intentionally NOT updated here.
		// They are only applied once Stripe confirms the change via the
		// customer.subscription.updated webhook, which derives the type from the
		// subscription items (requiring all items to belong to a single plan).

		org.SubscriptionCustomerOrganizationQty = customerOrgQty
		org.SubscriptionUserAccountQty = userAccountQty

		if err := db.UpdateOrganization(ctx, org); err != nil {
			return nil, fmt.Errorf("failed to update organization: %w", err)
		}

		if info, err = buildSubscriptionInfo(ctx, org); err != nil {
			return nil, fmt.Errorf("failed to build subscription info: %w", err)
		}
		return nil, nil
	}); resp != nil || err != nil {
		return resp, err
	} else {
		return respond.JSON(info), nil
	}
}

// validateSubscriptionQuantities validates that the requested quantities meet all requirements
func validateSubscriptionQuantities(
	subscriptionType types.SubscriptionType,
	customerOrgQty int64,
	userAccountQty int64,
	usage *currentUsageCounts,
) error {
	// Validate that at least one user account and one customer organization are requested
	if userAccountQty < 1 {
		return fmt.Errorf("user account quantity (%d) cannot be less than 1", userAccountQty)
	}

	if customerOrgQty < 1 {
		return fmt.Errorf("customer quantity (%d) cannot be less than 1", customerOrgQty)
	}

	// Validate that requested quantities meet current usage minimums
	if customerOrgQty < usage.customerOrganizationCount {
		return fmt.Errorf(
			"customer quantity (%d) cannot be less than current count (%d)",
			customerOrgQty,
			usage.customerOrganizationCount,
		)
	}

	if userAccountQty < usage.userAccountCount {
		return fmt.Errorf(
			"user account quantity (%d) cannot be less than current count (%d)",
			userAccountQty,
			usage.userAccountCount,
		)
	}

	// Validate that the subscription type limits can accommodate the requested quantities
	customerOrgLimit := subscription.GetCustomersPerOrganizationLimit(subscriptionType)
	if customerOrgLimit.IsExceeded(customerOrgQty) {
		return fmt.Errorf(
			"subscription type %v can have at most %v customers, but %v were requested",
			subscriptionType,
			customerOrgLimit,
			customerOrgQty,
		)
	}

	// Validate that the subscription type can accommodate current max users per customer
	usersPerCustomerLimit := subscription.GetUsersPerCustomerOrganizationLimit(subscriptionType)
	if usersPerCustomerLimit.IsExceeded(usage.maxUsersPerCustomer) {
		return fmt.Errorf(
			"subscription type %v allows at most %v users per customer, "+
				"but you currently have a customer with %v users",
			subscriptionType,
			usersPerCustomerLimit,
			usage.maxUsersPerCustomer,
		)
	}

	// Validate that the subscription type can accommodate current max deployments per customer
	deploymentsPerCustomerLimit := subscription.GetDeploymentTargetsPerCustomerOrganizationLimit(subscriptionType)
	if deploymentsPerCustomerLimit.IsExceeded(usage.maxDeploymentTargetsPerCustomer) {
		return fmt.Errorf(
			"subscription type %v allows at most %v deployment targets per customer, "+
				"but you currently have a customer with %v deployment targets",
			subscriptionType,
			deploymentsPerCustomerLimit,
			usage.maxDeploymentTargetsPerCustomer,
		)
	}

	if !subscriptionType.IsPro() {
		if usage.applicationEntitlementCount > 0 {
			return fmt.Errorf("subscription type %v does not allow application entitlements", subscriptionType)
		}
		if usage.artifactEntitlementCount > 0 {
			return fmt.Errorf("subscription type %v does not allow artifact entitlements", subscriptionType)
		}
	}

	return nil
}

// buildSubscriptionInfo builds the full subscription info response for an organization
func buildSubscriptionInfo(ctx context.Context, org *types.Organization) (*api.SubscriptionInfo, error) {
	// Get current usage counts
	usage, err := getCurrentUsageCounts(ctx, org.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage counts: %w", err)
	}

	// Check for entitlement management usage
	hasApplicationEntitlements, err := checkHasApplicationEntitlements(ctx, org.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check application entitlements: %w", err)
	}

	hasArtifactEntitlements, err := checkHasArtifactEntitlements(ctx, org.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check artifact entitlements: %w", err)
	}

	// Check for RBAC usage (non-admin roles)
	hasNonAdminRoles, err := checkHasNonAdminRoles(ctx, org.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check non-admin roles: %w", err)
	}

	hasAlertConfigurations, err := checkHasAlertConfigurations(
		ctx,
		org.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check alert configurations: %w", err)
	}

	info := &api.SubscriptionInfo{
		SubscriptionType:                    org.SubscriptionType,
		SubscriptionEndsAt:                  org.SubscriptionEndsAt,
		SubscriptionPeriod:                  org.SubscriptionPeriod,
		SubscriptionCustomerOrganizationQty: org.SubscriptionCustomerOrganizationQty.Value(),
		SubscriptionUserAccountQty:          org.SubscriptionUserAccountQty.Value(),
		CurrentUserAccountCount:             usage.userAccountCount,
		CurrentCustomerOrganizationCount:    usage.customerOrganizationCount,
		CurrentRegistryStorageBytes:         usage.registryStorageBytes,
		HasApplicationEntitlements:          hasApplicationEntitlements,
		HasArtifactEntitlements:             hasArtifactEntitlements,
		HasNonAdminRoles:                    hasNonAdminRoles,
		HasAlertConfigurations:              hasAlertConfigurations,
		Limits:                              map[types.SubscriptionType]api.SubscriptionLimits{},
	}

	for _, st := range types.AllSubscriptionTypes() {
		info.Limits[st] = subscription.GetSubscriptionLimits(st)
	}

	return info, nil
}

// currentUsageCounts represents the current usage counts for an organization
type currentUsageCounts struct {
	userAccountCount                int64
	customerOrganizationCount       int64
	maxUsersPerCustomer             int64
	maxDeploymentTargetsPerCustomer int64
	applicationEntitlementCount     int64
	artifactEntitlementCount        int64
	registryStorageBytes            int64
}

// getCurrentUsageCounts retrieves the current usage counts for the given organization
func getCurrentUsageCounts(ctx context.Context, orgID uuid.UUID) (*currentUsageCounts, error) {
	// Get current user account count
	userAccountCount, err := db.CountBillableUserAccountsByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user accounts: %w", err)
	}

	// Get current customer organization count
	customerOrgs, err := db.GetCustomerOrganizationsByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customers: %w", err)
	}

	appEntitlements, err := db.GetApplicationEntitlementsWithOrganizationID(ctx, orgID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get application entitlements: %w", err)
	}

	artifactEntitlements, err := db.GetArtifactEntitlements(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact entitlements: %w", err)
	}

	// The deployment targets per customer limit also applies to internal deployment targets
	maxDeploymentTargetsPerCustomer, err := db.CountDeploymentTargets(ctx, orgID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to count internal deployment targets: %w", err)
	}

	registryStorageBytes, err := db.GetRegistryStorageUsage(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry storage usage: %w", err)
	}

	// Find the maximum user count and deployment target count across all customer organizations
	var maxUsersPerCustomer int64
	for _, customerOrg := range customerOrgs {
		if customerOrg.UserCount > maxUsersPerCustomer {
			maxUsersPerCustomer = customerOrg.UserCount
		}
		if customerOrg.DeploymentTargetCount > maxDeploymentTargetsPerCustomer {
			maxDeploymentTargetsPerCustomer = customerOrg.DeploymentTargetCount
		}
	}

	return &currentUsageCounts{
		userAccountCount:                userAccountCount,
		customerOrganizationCount:       int64(len(customerOrgs)),
		maxUsersPerCustomer:             maxUsersPerCustomer,
		maxDeploymentTargetsPerCustomer: maxDeploymentTargetsPerCustomer,
		applicationEntitlementCount:     int64(len(appEntitlements)),
		artifactEntitlementCount:        int64(len(artifactEntitlements)),
		registryStorageBytes:            registryStorageBytes,
	}, nil
}

// checkHasApplicationEntitlements checks if the organization has any application entitlements
func checkHasApplicationEntitlements(ctx context.Context, orgID uuid.UUID) (bool, error) {
	entitlements, err := db.GetApplicationEntitlementsWithOrganizationID(ctx, orgID, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get application entitlements: %w", err)
	}
	return len(entitlements) > 0, nil
}

// checkHasArtifactEntitlements checks if the organization has any artifact entitlements
func checkHasArtifactEntitlements(ctx context.Context, orgID uuid.UUID) (bool, error) {
	hasEntitlement, err := db.HasAnyArtifactEntitlement(ctx, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to check artifact entitlements: %w", err)
	}
	return hasEntitlement, nil
}

// checkHasNonAdminRoles checks if the organization has any user accounts with non-admin roles
func checkHasNonAdminRoles(ctx context.Context, orgID uuid.UUID) (bool, error) {
	userAccounts, err := db.GetUserAccountsByOrgID(ctx, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to get user accounts: %w", err)
	}

	for _, user := range userAccounts {
		if user.UserRole != types.UserRoleAdmin {
			return true, nil
		}
	}
	return false, nil
}

func checkHasAlertConfigurations(ctx context.Context, orgID uuid.UUID) (bool, error) {
	alertConfigurationCount, err := db.CountAlertConfigurations(
		ctx,
		orgID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to get alert configurations count: %w", err)
	}

	return alertConfigurationCount > 0, nil
}
