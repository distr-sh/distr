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

type deploymentStatusNotificationKind int

const (
	deploymentStatusNotificationStale deploymentStatusNotificationKind = iota
	deploymentStatusNotificationStaleRecovered
	deploymentStatusNotificationError
	deploymentStatusNotificationErrorRecovered
)

// SendDeploymentStatusNotifications judges staleness by previousStatus, the newest status of any type, because
// progressing reports prove the agent alive and [RunDeploymentStatusNotifications] keys its warnings on the same
// status. Error transitions are judged by settledStatus (see [db.GetLatestSettledDeploymentRevisionStatus]).
func SendDeploymentStatusNotifications(
	ctx context.Context,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	previousStatus *types.DeploymentRevisionStatus,
	settledStatus *types.DeploymentRevisionStatus,
	currentStatus types.DeploymentRevisionStatus,
) error {
	log := internalctx.GetLogger(ctx).With(
		zap.String("currentStatus", string(currentStatus.Type)),
		zap.Time("currentStatusCreatedAt", currentStatus.CreatedAt))
	if previousStatus != nil {
		log = log.With(
			zap.String("previousStatus", string(previousStatus.Type)),
			zap.Time("previousStatusCreatedAt", previousStatus.CreatedAt),
		)
	}
	if settledStatus != nil {
		log = log.With(
			zap.String("settledStatus", string(settledStatus.Type)),
			zap.Time("settledStatusCreatedAt", settledStatus.CreatedAt),
		)
	}
	ctx = internalctx.WithLogger(ctx, log)

	kind, referenceStatus, ok := deploymentStatusNotificationFor(previousStatus, settledStatus, currentStatus)
	if !ok {
		log.Debug("notification not needed")
		return nil
	}

	configs, err := db.GetAlertConfigurationsForDeploymentTarget(ctx, deploymentTarget.ID)
	if err != nil {
		return err
	}

	for _, config := range configs {
		if err := sendDeploymentStatusNotificationsWithConfig(
			ctx, deploymentTarget, deployment, kind, referenceStatus, &currentStatus, config,
		); err != nil {
			return fmt.Errorf("failed to send deployment status notifications with config: %w", err)
		}
	}

	return nil
}

func RunDeploymentStatusNotifications(ctx context.Context) error {
	log := internalctx.GetLogger(ctx)

	log.Info("sending stale status notifications for all deployments")

	configs, err := db.GetAlertConfigurationsForAllOrganizations(ctx)
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
			deploymentTarget, err := db.GetDeploymentTarget(ctx, deploymentTargetID, nil, nil)
			if err != nil {
				return fmt.Errorf("failed to get deployment target: %w", err)
			}

			for _, deployment := range deploymentTarget.Deployments {
				log := log.With(zap.Stringer("deploymentId", deployment.ID))
				ctx := internalctx.WithLogger(ctx, log)
				newestStatus := deployment.NewestStatus()
				if newestStatus == nil {
					log.Debug("skip deployment with no status")
					continue
				}

				if !newestStatus.IsStale() {
					log.Debug("skip deployment with latest status not stale")
					continue
				}

				if err := sendDeploymentStatusNotificationsWithConfig(
					ctx, *deploymentTarget, deployment, deploymentStatusNotificationStale, newestStatus, nil, config,
				); err != nil {
					return fmt.Errorf("failed to send deployment status notifications with config: %w", err)
				}
			}
		}
	}

	log.Info("stale status notifications sent")

	return nil
}

// sendDeploymentStatusNotificationsWithConfig deduplicates notifications by the records stored for
// referenceStatus, so the caller has to pass the same status a matching earlier notification was keyed on.
func sendDeploymentStatusNotificationsWithConfig(
	ctx context.Context,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	kind deploymentStatusNotificationKind,
	referenceStatus *types.DeploymentRevisionStatus,
	currentStatus *types.DeploymentRevisionStatus,
	config types.AlertConfiguration,
) error {
	if !config.Enabled || !config.StatusTriggerEnabled {
		return nil
	}

	log := internalctx.GetLogger(ctx).With(zap.Stringer("configId", config.ID))

	organization, err := db.GetOrganizationByID(ctx, config.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	var existingRecord *types.NotificationRecord
	if referenceStatus != nil {
		existingRecord, err = db.GetLatestNotificationRecord(
			ctx, config.ID, referenceStatus.DeploymentRevisionID, referenceStatus.CreatedAt,
		)
		if err != nil && !errors.Is(err, apierrors.ErrNotFound) {
			return fmt.Errorf("failed to get latest notification record: %w", err)
		}
	}

	switch kind {
	case deploymentStatusNotificationStale:
		if existingRecord != nil {
			log.Debug("skip stale notifications because it was already sent")
			return nil
		}
	case deploymentStatusNotificationError, deploymentStatusNotificationErrorRecovered:
		if existingRecord != nil && existingRecord.CurrentStatusCreatedAt != nil {
			log.Debug("skip error/recovery notifications because it was already sent")
			return nil
		}
	case deploymentStatusNotificationStaleRecovered:
		if existingRecord == nil || existingRecord.CurrentStatusCreatedAt != nil {
			log.Debug("skip stale-recovery notifications because no unresolved stale notification was sent")
			return nil
		}
	}

	var aggErr error
	for _, user := range config.UserAccounts {
		log := log.With(zap.Stringer("userId", user.ID))
		log.Info("send notification")
		var err error
		switch kind {
		case deploymentStatusNotificationStale:
			err = mailsending.DeploymentStatusNotificationStale(
				ctx,
				user,
				*organization,
				deploymentTarget,
				deployment,
				*referenceStatus,
			)
		case deploymentStatusNotificationError:
			err = mailsending.DeploymentStatusNotificationError(
				ctx,
				user,
				*organization,
				deploymentTarget,
				deployment,
				*currentStatus,
			)
		default:
			err = mailsending.DeploymentStatusNotificationRecovered(
				ctx,
				user,
				*organization,
				deploymentTarget,
				deployment,
				*currentStatus,
			)
		}

		if err != nil {
			log.Warn("notification sending failed", zap.Error(err))
			aggErr = errors.Join(aggErr, err)
		}
	}

	recordType := types.NotificationRecordTypeResolved
	switch kind {
	case deploymentStatusNotificationStale:
		recordType = types.NotificationRecordTypeWarning
	case deploymentStatusNotificationError:
		recordType = types.NotificationRecordTypeAlert
	}

	record := types.NotificationRecord{
		OrganizationID:         config.OrganizationID,
		CustomerOrganizationID: config.CustomerOrganizationID,
		DeploymentTargetID:     &deploymentTarget.ID,
		AlertConfigurationID:   &config.ID,
		Type:                   recordType,
	}

	if currentStatus != nil {
		record.CurrentDeploymentRevisionID = &currentStatus.DeploymentRevisionID
		record.CurrentStatusCreatedAt = &currentStatus.CreatedAt
		record.CurrentStatusType = &currentStatus.Type
		record.CurrentStatusMessage = &currentStatus.Message
	}

	if referenceStatus != nil {
		record.PreviousDeploymentRevisionID = &referenceStatus.DeploymentRevisionID
		record.PreviousStatusCreatedAt = &referenceStatus.CreatedAt
	}

	if aggErr != nil {
		record.Message = aggErr.Error()
	}

	if err := db.SaveNotificationRecord(ctx, &record); err != nil {
		return fmt.Errorf("failed to save notification record: %w", err)
	}

	return nil
}

// deploymentStatusNotificationFor returns which notification currentStatus calls for and the status that
// notification is deduplicated by.
func deploymentStatusNotificationFor(
	previousStatus *types.DeploymentRevisionStatus,
	settledStatus *types.DeploymentRevisionStatus,
	currentStatus types.DeploymentRevisionStatus,
) (deploymentStatusNotificationKind, *types.DeploymentRevisionStatus, bool) {
	switch {
	case shouldNotifyError(previousStatus, settledStatus, currentStatus):
		return deploymentStatusNotificationError, settledStatus, true
	case shouldNotifyErrorRecovered(settledStatus, currentStatus):
		return deploymentStatusNotificationErrorRecovered, settledStatus, true
	case shouldNotifyStaleRecovered(previousStatus, currentStatus):
		return deploymentStatusNotificationStaleRecovered, previousStatus, true
	default:
		return 0, nil, false
	}
}

func shouldNotifyError(
	previousStatus *types.DeploymentRevisionStatus,
	settledStatus *types.DeploymentRevisionStatus,
	currentStatus types.DeploymentRevisionStatus,
) bool {
	return currentStatus.Type == types.DeploymentStatusTypeError &&
		(settledStatus == nil ||
			settledStatus.Type != types.DeploymentStatusTypeError ||
			previousStatus.IsStale())
}

func shouldNotifyStaleRecovered(
	previousStatus *types.DeploymentRevisionStatus,
	currentStatus types.DeploymentRevisionStatus,
) bool {
	return previousStatus.IsStale() && currentStatus.Type != types.DeploymentStatusTypeError
}

func shouldNotifyErrorRecovered(
	settledStatus *types.DeploymentRevisionStatus,
	currentStatus types.DeploymentRevisionStatus,
) bool {
	return settledStatus != nil &&
		settledStatus.Type == types.DeploymentStatusTypeError &&
		currentStatus.Type != types.DeploymentStatusTypeError &&
		currentStatus.Type != types.DeploymentStatusTypeProgressing
}
