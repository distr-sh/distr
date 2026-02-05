package subscription

import (
	"context"

	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/db"
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
		} else if _, err := db.DeleteDeploymentStatusNotificationConfigurationsWithOrganizationID(
			ctx,
			orgID,
		); err != nil {
			return err
		} else {
			return nil
		}
	})
}

func ReconcileStarterFeatures(ctx context.Context) error {
	return db.RunTx(ctx, func(ctx context.Context) error {
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
		} else {
			return nil
		}
	})
}
