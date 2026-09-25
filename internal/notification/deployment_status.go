package notification

import (
	"context"
	"errors"
	"fmt"
	"slices"

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
// progressing reports prove the agent alive. Error transitions are judged by settledStatus (see
// [db.GetLatestSettledDeploymentRevisionStatus]). Any report resolves the open stale warnings of the deployment, and
// an alert configuration whose warning it resolved gets a recovery notification unless the report calls for an error
// notification.
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

	resolvedConfigIDs, err := db.ResolveStaleWarnings(ctx, deployment.ID)
	if err != nil {
		return err
	}

	if _, ok := deploymentStatusNotificationFor(
		previousStatus, settledStatus, currentStatus, len(resolvedConfigIDs) > 0,
	); !ok {
		log.Debug("notification not needed")
		return nil
	}

	configs, err := db.GetAlertConfigurationsForDeploymentTarget(ctx, deploymentTarget.ID)
	if err != nil {
		return err
	}

	for _, config := range configs {
		kind, ok := deploymentStatusNotificationFor(
			previousStatus, settledStatus, currentStatus, slices.Contains(resolvedConfigIDs, config.ID),
		)
		if !ok {
			continue
		}
		if err := sendDeploymentStatusNotificationsWithConfig(
			ctx, deploymentTarget, deployment, kind, currentStatus, config,
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
					ctx, *deploymentTarget, deployment, deploymentStatusNotificationStale, *newestStatus, config,
				); err != nil {
					return fmt.Errorf("failed to send deployment status notifications with config: %w", err)
				}
			}
		}
	}

	log.Info("stale status notifications sent")

	return nil
}

// sendDeploymentStatusNotificationsWithConfig sends a notification about status, which is the stale status for a
// stale warning and the newly reported status otherwise.
func sendDeploymentStatusNotificationsWithConfig(
	ctx context.Context,
	deploymentTarget types.DeploymentTargetFull,
	deployment types.DeploymentWithLatestRevision,
	kind deploymentStatusNotificationKind,
	status types.DeploymentRevisionStatus,
	config types.AlertConfiguration,
) error {
	if !config.Enabled || !config.StatusTriggerEnabled {
		return nil
	}

	log := internalctx.GetLogger(ctx).With(zap.Stringer("configId", config.ID))

	if kind == deploymentStatusNotificationStale {
		if open, err := db.HasOpenStaleWarning(ctx, config.ID, deployment.ID); err != nil {
			return err
		} else if open {
			log.Debug("skip stale notifications because it was already sent")
			return nil
		}
	}

	organization, err := db.GetOrganizationByID(ctx, config.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
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
				status,
			)
		case deploymentStatusNotificationError:
			err = mailsending.DeploymentStatusNotificationError(
				ctx,
				user,
				*organization,
				deploymentTarget,
				deployment,
				status,
			)
		default:
			err = mailsending.DeploymentStatusNotificationRecovered(
				ctx,
				user,
				*organization,
				deploymentTarget,
				deployment,
				status,
			)
		}

		if err != nil {
			log.Warn("notification sending failed", zap.Error(err))
			aggErr = errors.Join(aggErr, err)
		}
	}

	recordType := types.NotificationRecordTypeResolved
	summary := status.Message
	switch kind {
	case deploymentStatusNotificationStale:
		recordType = types.NotificationRecordTypeWarning
		summary = "Stale"
	case deploymentStatusNotificationError:
		recordType = types.NotificationRecordTypeAlert
	}

	record := types.NotificationRecord{
		OrganizationID:         config.OrganizationID,
		CustomerOrganizationID: config.CustomerOrganizationID,
		SourceType:             types.NotificationSourceTypeAlert,
		SourceConfigurationID:  &config.ID,
		SubjectID:              &deploymentTarget.ID,
		Type:                   recordType,
		DeploymentRevisionID:   &status.DeploymentRevisionID,
		Details: types.NotificationRecordDetails{
			Summary:                  summary,
			CustomerOrganizationName: customerOrganizationName(deploymentTarget),
			DeploymentTargetName:     &deploymentTarget.Name,
			ApplicationName:          &deployment.Application.Name,
			ApplicationType:          &deployment.Application.Type,
			ApplicationVersionName:   new(statusApplicationVersionName(deployment, status)),
		},
	}

	if aggErr != nil {
		record.DeliveryError = aggErr.Error()
	}

	if err := db.SaveNotificationRecord(ctx, &record); err != nil {
		return fmt.Errorf("failed to save notification record: %w", err)
	}

	return nil
}

// statusApplicationVersionName names the version of the revision that reported status, which is the
// applied one rather than the latest one while a newer revision is still being rolled out.
func statusApplicationVersionName(
	deployment types.DeploymentWithLatestRevision,
	status types.DeploymentRevisionStatus,
) string {
	if deployment.CurrentDeploymentRevisionID != nil &&
		*deployment.CurrentDeploymentRevisionID == status.DeploymentRevisionID &&
		deployment.CurrentApplicationVersionName != nil {
		return *deployment.CurrentApplicationVersionName
	}
	return deployment.ApplicationVersionName
}

// deploymentStatusNotificationFor returns which notification currentStatus calls for. staleWarningResolved tells
// whether the report resolved an open stale warning of the alert configuration.
func deploymentStatusNotificationFor(
	previousStatus *types.DeploymentRevisionStatus,
	settledStatus *types.DeploymentRevisionStatus,
	currentStatus types.DeploymentRevisionStatus,
	staleWarningResolved bool,
) (deploymentStatusNotificationKind, bool) {
	switch {
	case shouldNotifyError(previousStatus, settledStatus, currentStatus):
		return deploymentStatusNotificationError, true
	case shouldNotifyErrorRecovered(settledStatus, currentStatus):
		return deploymentStatusNotificationErrorRecovered, true
	case staleWarningResolved && currentStatus.Type != types.DeploymentStatusTypeError:
		return deploymentStatusNotificationStaleRecovered, true
	default:
		return 0, false
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

func shouldNotifyErrorRecovered(
	settledStatus *types.DeploymentRevisionStatus,
	currentStatus types.DeploymentRevisionStatus,
) bool {
	return settledStatus != nil &&
		settledStatus.Type == types.DeploymentStatusTypeError &&
		currentStatus.Type != types.DeploymentStatusTypeError &&
		currentStatus.Type != types.DeploymentStatusTypeProgressing
}
