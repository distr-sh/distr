package subscription

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/license"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

const GlobalOrganizationLimitReachedMessage = "global organization limit has been reached"

// ErrGlobalOrganizationLimitReached is returned instead of creating an organization when the license's global
// organization limit has been reached. Organization-less users run into it by design once the limit is hit, so
// it is a bad request that callers report to the user rather than an unexpected error.
var ErrGlobalOrganizationLimitReached = apierrors.NewBadRequest(GlobalOrganizationLimitReachedMessage)

// IsGlobalOrganizationLimitReached reports whether the license's global limit on the total number of
// organizations across the whole instance has been reached, meaning no further organization may be created
// (including a personal organization auto-created for a new user). It returns false when the license does
// not define a global limit.
func IsGlobalOrganizationLimitReached(ctx context.Context) (bool, error) {
	limit := license.GetLicenseData().MaxOrganizations
	if limit.IsUnlimited() {
		return false, nil
	} else if count, err := db.CountAllOrganizations(ctx); err != nil {
		return false, fmt.Errorf("could not query Organization: %w", err)
	} else {
		return limit.IsReached(count), nil
	}
}

func IsBillableUserAccountLimitReached(ctx context.Context, org types.Organization) (bool, error) {
	if !org.HasActiveSubscription() {
		return true, nil
	} else if org.SubscriptionUserAccountQty.IsUnlimited() {
		return false, nil
	} else if count, err := db.CountBillableUserAccountsByOrgID(ctx, org.ID); err != nil {
		return true, err
	} else {
		return org.SubscriptionUserAccountQty.IsReached(count), nil
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
	} else if count, err := db.CountDeploymentTargets(ctx, org.ID, customerOrgID); err != nil {
		return true, fmt.Errorf("could not query DeploymentTarget: %w", err)
	} else {
		return GetDeploymentTargetsPerCustomerOrganizationLimit(org.SubscriptionType).IsReached(count), nil
	}
}
