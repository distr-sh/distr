package subscription

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/license"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func ReconcileStarterFeaturesForOrganizationID(ctx context.Context, orgID uuid.UUID) error {
	return db.RunTx(ctx, func(ctx context.Context) error {
		if err := db.UpdateAllUserAccountOrganizationAssignmentsWithOrganizationID(
			ctx,
			orgID,
			types.UserRoleAdmin,
		); err != nil {
			return err
		} else if err := db.UpdateDeploymentUnsetLicenseIDWithOrganizationID(ctx, orgID); err != nil {
			return err
		} else if _, err := db.DeleteApplicationLicensesWithOrganizationID(ctx, orgID); err != nil {
			return err
		} else if _, err := db.DeleteArtifactLicensesWithOrganizationID(ctx, orgID); err != nil {
			return err
		} else if _, err := db.DeleteAlertConfigurationsWithOrganizationID(ctx, orgID); err != nil {
			return err
		} else {
			return nil
		}
	})
}

func ReconcileStarterFeatures(ctx context.Context) error {
	return db.RunTx(ctx, func(ctx context.Context) error {
		licenseData := license.GetLicenseData()

		if buildconfig.IsCommunityEdition() {
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
		} else if err := db.UpdateDeploymentUnsetLicenseIDWithOrganizationSubscriptionType(
			ctx,
			types.NonProSubscriptionTypes,
		); err != nil {
			return err
		} else if _, err := db.DeleteApplicationLicensesWithOrganizationSubscriptionType(
			ctx,
			types.NonProSubscriptionTypes,
		); err != nil {
			return err
		} else if _, err := db.DeleteArtifactLicensesWithOrganizationSubscriptionType(
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
			if err := db.UpdateOrganizationEnterpriseLimits(
				ctx,
				licenseData.MaxCustomersPerOrganization,
				licenseData.MaxUsersPerOrganization,
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
