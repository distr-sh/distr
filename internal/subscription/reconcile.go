package subscription

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/license"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func ReconcileStarterFeaturesForOrganizationID(ctx context.Context, log *zap.Logger, orgID uuid.UUID) error {
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

func ReconcileEditionFeatures(ctx context.Context, log *zap.Logger) error {
	log.Info("reconciling edition features")
	return db.RunTx(ctx, func(ctx context.Context) error {
		parsedLicense := license.GetParsedLicense()

		if buildconfig.IsCommunityEdition() {
			log.Info("updating organization subscription type to community")
			if err := db.UpdateOrganizationSubscriptionType(ctx, types.SubscriptionTypeCommunity); err != nil {
				return err
			}
		}

		if buildconfig.IsEnterpriseEdition() {
			log.Info("updating organization subscription type to enterprise")
			if err := db.UpdateOrganizationSubscriptionType(ctx, types.SubscriptionTypeEnterprise); err != nil {
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

		if parsedLicense.EnforceLimitsOnStartup {
			log.Info("updating enterprise edition limits",
				zap.Any("max_customers", parsedLicense.MaxCustomersPerOrganization),
				zap.Any("max_users", parsedLicense.MaxUsersPerOrganization),
				zap.String("subscription_period", string(parsedLicense.Period)),
				zap.Time("subscription_ends_at", parsedLicense.ExpirationDate),
			)
			if err := db.UpdateOrganizationEnterpriseLimits(
				ctx,
				parsedLicense.MaxCustomersPerOrganization,
				parsedLicense.MaxUsersPerOrganization,
				parsedLicense.Period,
				parsedLicense.ExpirationDate,
			); err != nil {
				return err
			}

			if limit := parsedLicense.MaxOrganizations; !limit.IsUnlimited() {
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
