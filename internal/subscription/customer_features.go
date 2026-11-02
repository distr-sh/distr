package subscription

import (
	"context"
	"fmt"
	"slices"

	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

// GrantCustomerFeatures gives every customer of org the customer features of the organization
// features org has gained since previousFeatures (see [types.GrantedCustomerOrganizationFeatures]).
func GrantCustomerFeatures(ctx context.Context, org *types.Organization, previousFeatures []types.Feature) error {
	for _, feature := range types.GainedCustomerOrganizationFeatures(previousFeatures, org.Features) {
		if err := db.AddCustomerOrganizationFeature(ctx, []uuid.UUID{org.ID}, feature); err != nil {
			return fmt.Errorf("failed to grant %v to customers: %w", feature, err)
		}
	}
	return nil
}

// applyPlanToAllOrganizations applies the plan with [db.ApplyPlanToAllOrganizations] and gives the
// customers of every organization that gains a feature through it the matching customer features.
func applyPlanToAllOrganizations(ctx context.Context, plan types.SubscriptionPlan) error {
	planFeatures := plan.Features()
	gaining := make(map[types.CustomerOrganizationFeature][]uuid.UUID)
	for _, granted := range types.GrantedCustomerOrganizationFeatures {
		if !slices.Contains(planFeatures, granted.Feature) {
			continue
		} else if ids, err := db.GetOrganizationIDsWithoutFeature(ctx, granted.Feature); err != nil {
			return err
		} else {
			gaining[granted.CustomerFeature] = ids
		}
	}

	if err := db.ApplyPlanToAllOrganizations(ctx, plan); err != nil {
		return err
	}

	for customerFeature, ids := range gaining {
		if err := db.AddCustomerOrganizationFeature(ctx, ids, customerFeature); err != nil {
			return fmt.Errorf("failed to grant %v to customers: %w", customerFeature, err)
		}
	}
	return nil
}
