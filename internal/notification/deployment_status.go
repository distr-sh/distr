package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/types"
	"go.uber.org/zap"
)

func SendDeploymentStatusNotifications(
	ctx context.Context,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	previousStatus types.DeploymentRevisionStatus,
	currentStatus types.DeploymentRevisionStatus,
) error {
	log := internalctx.GetLogger(ctx).With(
		zap.String("previousStatus", string(previousStatus.Type)),
		zap.Time("previousStatusCreatedAt", previousStatus.CreatedAt),
		zap.String("currentStatus", string(currentStatus.Type)),
		zap.Time("currentStatusCreatedAt", currentStatus.CreatedAt))
	ctx = internalctx.WithLogger(ctx, log)

	if !shouldNotify(previousStatus, currentStatus) {
		log.Debug("notification not needed")
		return nil
	}

	configs, err := db.GetDeploymentStatusNotificationConfigurationsForDeploymentTarget(ctx, deploymentTarget.ID)
	if err != nil {
		return err
	}

	for _, config := range configs {
		if err := sendDeploymentStatusNotificationsWithConfig(
			ctx, deploymentTarget, deployment, previousStatus, &currentStatus, config,
		); err != nil {
			return fmt.Errorf("failed to send deployment status notifications with config: %w", err)
		}
	}

	return nil
}

func RunDeploymentStatusNotifications(ctx context.Context) error {
	log := internalctx.GetLogger(ctx)

	log.Info("sending stale status notifications for all deployments")

	configs, err := db.GetDeploymentStatusNotificationConfigurationsForAllOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all configs: %w", err)
	}

	for _, config := range configs {
		log := log.With(zap.Stringer("configId", config.ID))
		if !config.Enabled {
			log.Debug("skip disabled config")
			continue
		}

		for _, deploymentTargetID := range config.DeploymentTargetIDs {
			log := log.With(zap.Stringer("deploymentTargetId", deploymentTargetID))
			deploymentTarget, err := db.GetDeploymentTarget(ctx, deploymentTargetID, nil)
			if err != nil {
				return fmt.Errorf("failed to get deployment target: %w", err)
			}

			for _, deployment := range deploymentTarget.Deployments {
				log := log.With(zap.Stringer("deploymentId", deployment.ID))
				ctx := internalctx.WithLogger(ctx, log)
				if deployment.LatestStatus == nil {
					log.Debug("skip deployment with no status")
					continue
				}

				if !deployment.LatestStatus.IsStale() {
					log.Debug("skip deployment with latest status not stale")
					continue
				}

				if err := sendDeploymentStatusNotificationsWithConfig(
					ctx, *deploymentTarget, deployment, *deployment.LatestStatus, nil, config,
				); err != nil {
					return fmt.Errorf("failed to send deployment status notifications with config: %w", err)
				}
			}
		}
	}

	log.Info("stale status notifications sent")

	return nil
}

func sendDeploymentStatusNotificationsWithConfig(
	ctx context.Context,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	previousStatus types.DeploymentRevisionStatus,
	currentStatus *types.DeploymentRevisionStatus,
	config types.DeploymentStatusNotificationConfiguration,
) error {
	if !config.Enabled {
		return nil
	}

	log := internalctx.GetLogger(ctx).With(zap.Stringer("configId", config.ID))

	existingRecord, err := db.GetLatestNotificationRecord(ctx, config.ID, previousStatus.ID)
	if err != nil && !errors.Is(err, apierrors.ErrNotFound) {
		return fmt.Errorf("failed to get latest notification record: %w", err)
	}

	if currentStatus == nil {
		if existingRecord != nil {
			log.Debug("skip stale notifications because it was already sent")
			return nil
		}
	} else if shouldNotifyError(previousStatus, *currentStatus) ||
		shouldNotifyErrorRecovered(previousStatus, *currentStatus) {
		if existingRecord != nil && existingRecord.CurrentDeploymentRevisionStatusID != nil {
			log.Debug("skip error/recovery notifications because it was already sent")
			return nil
		}
	} else {
		if existingRecord == nil {
			log.Debug("skip stale-recovery notifications because no previous stale notification was sent")
			return nil
		}
	}

	var aggErr error
	for _, user := range config.UserAccounts {
		log := log.With(zap.Stringer("userId", user.ID))
		log.Info("send notification")
		var err error
		if currentStatus == nil {
			err = mailsending.DeploymentStatusNotificationStale(ctx, user, deploymentTarget, deployment, previousStatus)
		} else if currentStatus.Type == types.DeploymentStatusTypeError {
			err = mailsending.DeploymentStatusNotificationError(ctx, user, deploymentTarget, deployment, *currentStatus)
		} else {
			err = mailsending.DeploymentStatusNotificationRecovered(ctx, user, deploymentTarget, deployment, *currentStatus)
		}

		if err != nil {
			log.Warn("notification sending failed", zap.Error(err))
			aggErr = errors.Join(aggErr, err)
		}
	}

	record := types.NotificationRecord{
		OrganizationID:                              deploymentTarget.OrganizationID,
		CustomerOrganizationID:                      deploymentTarget.CustomerOrganizationID,
		DeploymentTargetID:                          &deploymentTarget.ID,
		DeploymentStatusNotificationConfigurationID: &config.ID,
		PreviousDeploymentRevisionStatusID:          &previousStatus.ID,
	}

	if currentStatus != nil {
		record.CurrentDeploymentRevisionStatusID = &currentStatus.ID
	}

	if aggErr != nil {
		record.Message = aggErr.Error()
	}

	if err := db.SaveNotificationRecord(ctx, &record); err != nil {
		return fmt.Errorf("failed to save notification record: %w", err)
	}

	return nil
}

func shouldNotifyError(previousStatus, currentStatus types.DeploymentRevisionStatus) bool {
	return (previousStatus.Type != types.DeploymentStatusTypeError || previousStatus.IsStale()) &&
		currentStatus.Type == types.DeploymentStatusTypeError
}

func shouldNotifyStaleRecovered(previousStatus, currentStatus types.DeploymentRevisionStatus) bool {
	return previousStatus.IsStale() && currentStatus.Type != types.DeploymentStatusTypeError
}

func shouldNotifyErrorRecovered(previousStatus, currentStatus types.DeploymentRevisionStatus) bool {
	return previousStatus.Type == types.DeploymentStatusTypeError && currentStatus.Type != types.DeploymentStatusTypeError
}

func shouldNotify(previousStatus, currentStatus types.DeploymentRevisionStatus) bool {
	return shouldNotifyError(previousStatus, currentStatus) ||
		shouldNotifyStaleRecovered(previousStatus, currentStatus) ||
		shouldNotifyErrorRecovered(previousStatus, currentStatus)
}
