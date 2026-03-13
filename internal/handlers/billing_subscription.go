package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/billing"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v84"
	"go.uber.org/zap"
)

func GetSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	org := auth.CurrentOrg()

	info, err := buildSubscriptionInfo(ctx, org)
	if err != nil {
		http.Error(w, "failed to build subscription info", http.StatusInternalServerError)
		return
	}

	RespondJSON(w, info)
}

func CreateSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	var body struct {
		SubscriptionType        types.SubscriptionType   `json:"subscriptionType"`
		SubscriptionPeriod      types.SubscriptionPeriod `json:"subscriptionPeriod"`
		CustomerOrganizationQty int64                    `json:"subscriptionCustomerOrganizationQuantity"`
		UserAccountQty          int64                    `json:"subscriptionUserAccountQuantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Debug("bad json payload", zap.Error(err))
		http.Error(w, "bad json payload", http.StatusBadRequest)
		return
	}

	// Get current usage counts
	usage, err := getCurrentUsageCounts(ctx, *auth.CurrentOrgID())
	if err != nil {
		log.Error("failed to get current usage counts", zap.Error(err))
		http.Error(w, "failed to get current usage counts", http.StatusInternalServerError)
		return
	}

	// Validate subscription quantities
	if err := validateSubscriptionQuantities(
		body.SubscriptionType,
		body.CustomerOrganizationQty,
		body.UserAccountQty,
		usage,
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session, err := billing.CreateCheckoutSession(ctx, billing.CheckoutSessionParams{
		OrganizationID:          auth.CurrentOrgID().String(),
		TrialEndsAt:             auth.CurrentOrg().SubscriptionEndsAt,
		SubscriptionType:        body.SubscriptionType,
		SubscriptionPeriod:      body.SubscriptionPeriod,
		CustomerOrganizationQty: body.CustomerOrganizationQty,
		UserAccountQty:          body.UserAccountQty,
		Currency:                "usd",
		SuccessURL:              fmt.Sprintf("%v/subscription/callback", handlerutil.GetRequestSchemeAndHost(r)),
	})
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("failed to create checkout session", zap.Error(err))
		http.Error(w, "failed to create checkout session", http.StatusInternalServerError)
		return
	}

	RespondJSON(w, api.CheckoutResponse{
		SessionID: session.ID,
		URL:       session.URL,
	})
}

func UpdateSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	org := auth.CurrentOrg()

	// Check if organization has an active subscription
	if org.SubscriptionType == types.SubscriptionTypeTrial {
		http.Error(w, "cannot update trial subscription, please create a new subscription", http.StatusBadRequest)
		return
	}

	if org.StripeSubscriptionID == nil || *org.StripeSubscriptionID == "" {
		http.Error(w, "no active subscription found", http.StatusBadRequest)
		return
	}

	var body struct {
		CustomerOrganizationQty int64 `json:"subscriptionCustomerOrganizationQuantity"`
		UserAccountQty          int64 `json:"subscriptionUserAccountQuantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Debug("bad json payload", zap.Error(err))
		http.Error(w, "bad json payload", http.StatusBadRequest)
		return
	}

	var info *api.SubscriptionInfo
	shouldRespond := false
	err := db.RunTxRR(ctx, func(ctx context.Context) error {
		org, err := db.GetOrganizationByID(ctx, org.ID)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to get organization", zap.Error(err))
			http.Error(w, "failed to get organization", http.StatusInternalServerError)
			return err
		}

		usage, err := getCurrentUsageCounts(ctx, org.ID)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to get current usage counts", zap.Error(err))
			http.Error(w, "failed to get current usage counts", http.StatusInternalServerError)
			return err
		}

		if err := validateSubscriptionQuantities(
			org.SubscriptionType,
			body.CustomerOrganizationQty,
			body.UserAccountQty,
			usage,
		); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}

		updatedSub, err := billing.UpdateSubscription(ctx, billing.SubscriptionUpdateParams{
			SubscriptionID:          *org.StripeSubscriptionID,
			CustomerOrganizationQty: body.CustomerOrganizationQty,
			UserAccountQty:          body.UserAccountQty,
			ReturnURL:               fmt.Sprintf("%v/subscription", handlerutil.GetRequestSchemeAndHost(r)),
		})
		if err != nil {
			log.Error("failed to update subscription", zap.Error(err))

			if stripeErr := new(stripe.Error); errors.As(err, &stripeErr) {
				if stripeErr.Type == stripe.ErrorTypeInvalidRequest {
					if stripeErr.Code == stripe.ErrorCodeResourceMissing {
						http.Error(w, "stripe error: subscription not found", http.StatusBadRequest)
						return err
					} else {
						http.Error(w, "stripe error: invalid request (has the subscription been canceled?)", http.StatusBadRequest)
						return err
					}
				}
			}

			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, "failed to update subscription", http.StatusInternalServerError)
			return err
		}

		customerOrgQty, err := billing.GetCustomerOrganizationQty(*updatedSub)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to get customer quantity from updated subscription", zap.Error(err))
			http.Error(w, "failed to get customer quantity", http.StatusInternalServerError)
			return err
		}

		userAccountQty, err := billing.GetUserAccountQty(*updatedSub)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to get user account quantity from updated subscription", zap.Error(err))
			http.Error(w, "failed to get user account quantity", http.StatusInternalServerError)
			return err
		}

		org.SubscriptionCustomerOrganizationQty = customerOrgQty
		org.SubscriptionUserAccountQty = userAccountQty

		if err := db.UpdateOrganization(ctx, org); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to update organization", zap.Error(err))
			http.Error(w, "failed to update organization", http.StatusInternalServerError)
			return err
		}

		info, err = buildSubscriptionInfo(ctx, org)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("failed to build subscription info", zap.Error(err))
			http.Error(w, "failed to build subscription info", http.StatusInternalServerError)
			return err
		}

		shouldRespond = true

		return nil
	})

	if err != nil {
		log.Error("update subscription failed", zap.Error(err))
		if shouldRespond {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	} else {
		RespondJSON(w, info)
	}
}

// validateSubscriptionQuantities validates that the requested quantities meet all requirements
func validateSubscriptionQuantities(
	subscriptionType types.SubscriptionType,
	customerOrgQty int64,
	userAccountQty int64,
	usage *currentUsageCounts,
) error {
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
		SubscriptionType:                       org.SubscriptionType,
		SubscriptionEndsAt:                     org.SubscriptionEndsAt,
		SubscriptionPeriod:                     org.SubscriptionPeriod,
		SubscriptionCustomerOrganizationQty:    org.SubscriptionCustomerOrganizationQty.Value(),
		SubscriptionUserAccountQty:             org.SubscriptionUserAccountQty.Value(),
		CurrentUserAccountCount:                usage.userAccountCount,
		CurrentCustomerOrganizationCount:       usage.customerOrganizationCount,
		CurrentMaxUsersPerCustomer:             usage.maxUsersPerCustomer,
		CurrentMaxDeploymentTargetsPerCustomer: usage.maxDeploymentTargetsPerCustomer,
		HasApplicationEntitlements:             hasApplicationEntitlements,
		HasArtifactEntitlements:                hasArtifactEntitlements,
		HasNonAdminRoles:                       hasNonAdminRoles,
		HasAlertConfigurations:                 hasAlertConfigurations,
		Limits:                                 map[types.SubscriptionType]api.SubscriptionLimits{},
	}

	for _, st := range types.AllSubscriptionTypes {
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
}

// getCurrentUsageCounts retrieves the current usage counts for the given organization
func getCurrentUsageCounts(ctx context.Context, orgID uuid.UUID) (*currentUsageCounts, error) {
	// Get current user account count
	userAccountCount, err := db.CountVendorUserAccountsByOrgID(ctx, orgID)
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

	// Find the maximum user count and deployment target count across all customer organizations
	var maxUsersPerCustomer int64
	var maxDeploymentTargetsPerCustomer int64
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
