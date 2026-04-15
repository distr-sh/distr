package subscription

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/license"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func ReconcileStarterFeaturesForOrganizationID(ctx context.Context, orgID uuid.UUID) error {
	log := internalctx.GetLogger(ctx)
	log.Info("reconciling starter features for organization", zap.String("organization_id", orgID.String()))
	return db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.UpdateAllUserAccountOrganizationAssignmentsWithOrganizationID(
			ctx,
			orgID,
			types.UserRoleAdmin,
		); err != nil {
			return err
		} else if err := db.UpdateDeploymentUnsetEntitlementIDWithOrganizationID(ctx, orgID); err != nil {
			return err
		} else if _, err := db.DeleteApplicationEntitlementsWithOrganizationID(ctx, orgID); err != nil {
			return err
		} else if _, err := db.DeleteArtifactEntitlementsWithOrganizationID(ctx, orgID); err != nil {
			return err
		} else if _, err := db.DeleteAlertConfigurationsWithOrganizationID(ctx, orgID); err != nil {
			return err
		} else {
			return nil
		}
	})
}

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
		} else if err := db.UpdateOrganizationFeaturesWithSubscriptionType(
			ctx,
			types.NonProSubscriptionTypes,
			[]types.Feature{},
		); err != nil {
			return err
		}

		if licenseData.EnforceLimitsOnStartup {
			log.Info("updating enterprise edition limits",
				zap.Any("max_customers", licenseData.MaxCustomersPerOrganization),
				zap.Any("max_users", licenseData.MaxUsersPerOrganization),
				zap.String("subscription_period", string(licenseData.Period)),
				zap.Time("subscription_ends_at", licenseData.ExpirationDate),
			)
			if err := db.UpdateOrganizationEnterpriseLimits(
				ctx,
				licenseData.MaxCustomersPerOrganization,
				licenseData.MaxUsersPerOrganization,
				licenseData.Period,
				licenseData.ExpirationDate,
			); err != nil {
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
