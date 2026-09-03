package subscription

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/license"
	"github.com/distr-sh/distr/internal/types"
	"go.uber.org/zap"
)

func ReconcileEditionFeatures(ctx context.Context) error {
	log := internalctx.GetLogger(ctx)
	log.Info("reconciling edition features")
	return db.RunTx(ctx, func(ctx context.Context) error {
		licenseData := license.GetLicenseData()

		if buildconfig.IsCommunityEdition() {
			log.Info("updating organization subscription type to community")
			if err := db.UpdateOrganizationSubscriptionType(ctx, types.SubscriptionTypeCommunity); err != nil {
				return err
			}
		}

		if err := db.UpdateAllUserAccountOrganizationAssignmentsWithOrganizationSuscriptionType(
			ctx,
			types.NonProSubscriptionTypes,
			types.UserRoleAdmin,
		); err != nil {
			return err
		} else if err := db.UpdateDeploymentUnsetEntitlementIDWithOrganizationSubscriptionType(
			ctx,
			types.NonProSubscriptionTypes,
		); err != nil {
			return err
		} else if _, err := db.DeleteApplicationEntitlementsWithOrganizationSubscriptionType(
			ctx,
			types.NonProSubscriptionTypes,
		); err != nil {
			return err
		} else if _, err := db.DeleteArtifactEntitlementsWithOrganizationSubscriptionType(
			ctx,
			types.NonProSubscriptionTypes,
		); err != nil {
			return err
		} else if err := db.RemoveOrganizationFeaturesWithSubscriptionType(
			ctx,
			types.NonProSubscriptionTypes,
			types.PlanManagedFeatures,
		); err != nil {
			return err
		}

		if licenseData.EnforceLimitsOnStartup {
			plan := licenseData.Plan()
			log.Info("applying the licensed plan to all organizations",
				zap.String("subscription_type", string(plan.Type)),
				zap.Any("features", plan.Features()),
				zap.Any("max_customers", plan.CustomerOrganizationQty),
				zap.Any("max_users", plan.UserAccountQty),
				zap.String("subscription_period", string(plan.Period)),
				zap.Time("subscription_ends_at", plan.EndsAt),
			)
			if err := db.ApplyPlanToAllOrganizations(ctx, plan); err != nil {
				return err
			}

			if limit := licenseData.MaxOrganizations; !limit.IsUnlimited() {
				if count, err := db.CountAllOrganizations(ctx); err != nil {
					return err
				} else if limit.IsExceeded(count) {
					return fmt.Errorf("global organizations count is exceeded (limit: %v, got %v)", limit, count)
				} else {
					return nil
				}
			}
		}

		return nil
	})
}
