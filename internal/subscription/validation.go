package subscription

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func IsVendorUserAccountLimitReached(ctx context.Context, org types.Organization) (bool, error) {
	if !org.HasActiveSubscription() {
		return true, nil
	} else if org.SubscriptionUserAccountQty.IsUnlimited() {
		return false, nil
	} else if vendorCount, err := db.CountVendorUserAccountsByOrgID(ctx, org.ID); err != nil {
		return true, err
	} else {
		return org.SubscriptionUserAccountQty.IsReached(vendorCount), nil
	}
}

func IsCustomerUserAccountLimitReached(
	org types.Organization,
	customerOrganization types.CustomerOrganizationWithUsage,
) (bool, error) {
	if !org.HasActiveSubscription() {
		return true, nil
	} else {
		return GetUsersPerCustomerOrganizationLimit(org.SubscriptionType).IsReached(customerOrganization.UserCount),
			nil
	}
}

func IsCustomerOrganizationLimitReached(ctx context.Context, org types.Organization) (bool, error) {
	if !org.HasActiveSubscription() {
		return true, nil
	} else if org.SubscriptionCustomerOrganizationQty.IsUnlimited() {
		return false, nil
	} else {
		if customerOrgCount, err := db.CountCustomerOrganizationsByOrganizationID(ctx, org.ID); err != nil {
			return true, fmt.Errorf("could not query CustomerOrganization: %w", err)
		} else {
			return org.SubscriptionCustomerOrganizationQty.IsReached(customerOrgCount), nil
		}
	}
}

func IsDeploymentTargetLimitReached(
	ctx context.Context,
	org types.Organization,
	customerOrgID *uuid.UUID,
) (bool, error) {
	if !org.HasActiveSubscription() {
		return true, nil
	} else if org.SubscriptionType == types.SubscriptionTypeTrial {
		return false, nil
	} else if count, err := db.CountDeploymentTargets(ctx, org.ID, customerOrgID); err != nil {
		return true, fmt.Errorf("could not query DeploymentTarget: %w", err)
	} else {
		return GetDeploymentTargetsPerCustomerOrganizationLimit(org.SubscriptionType).IsReached(count), nil
	}
}
