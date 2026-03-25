package notification

import (
	"context"
	"errors"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mailsending"
	"github.com/distr-sh/distr/internal/types"
	"go.uber.org/zap"
)

func SendDeploymentTargetMetricsNotifications(
	ctx context.Context,
	deploymentTarget types.DeploymentTargetFull,
	previousMetrics *types.DeploymentTargetMetrics,
	currentMetrics types.DeploymentTargetMetrics,
) error {
	if !deploymentTarget.MetricsEnabled {
		return nil
	}

	configs, err := db.GetAlertConfigurationsForDeploymentTarget(ctx, deploymentTarget.ID)
	if err != nil {
		return err
	}

	for _, config := range configs {
		if err := sendDeploymentTargetMetricsNotificationsWithConfig(
			ctx, deploymentTarget, previousMetrics, currentMetrics, config,
		); err != nil {
			return fmt.Errorf("failed to send deployment target metrics notifications with config: %w", err)
		}
	}

	return nil
}

func sendDeploymentTargetMetricsNotificationsWithConfig(
	ctx context.Context,
	deploymentTarget types.DeploymentTargetFull,
	previousMetrics *types.DeploymentTargetMetrics,
	currentMetrics types.DeploymentTargetMetrics,
	config types.AlertConfiguration,
) error {
	if !config.Enabled || !config.AnyThresholdEnabled() {
		return nil
	}

	log := internalctx.GetLogger(ctx).With(zap.Stringer("configId", config.ID))

	organization, err := db.GetOrganizationByID(ctx, config.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	if shouldNotifyResource(config.CpuTriggerThreshold, previousMetrics, currentMetrics, cpuUsage) {
		log.Info("sending CPU alert notification")
		if err := sendMetricNotification(
			ctx, false, deploymentTarget, *organization, config, previousMetrics, currentMetrics,
			"cpu", "", "", *config.CpuTriggerThreshold, int(currentMetrics.CPUUsage*100),
		); err != nil {
			return err
		}
	} else if shouldNotifyResourceResolved(config.CpuTriggerThreshold, previousMetrics, currentMetrics, cpuUsage) {
		log.Info("sending CPU alert resolved notification")
		if err := sendMetricNotification(
			ctx, true, deploymentTarget, *organization, config, previousMetrics, currentMetrics,
			"cpu", "", "", *config.CpuTriggerThreshold, int(currentMetrics.CPUUsage*100),
		); err != nil {
			return err
		}
	}

	if shouldNotifyResource(config.MemoryTriggerThreshold, previousMetrics, currentMetrics, memoryUsage) {
		log.Info("sending memory alert notification")
		if err := sendMetricNotification(
			ctx, false, deploymentTarget, *organization, config, previousMetrics, currentMetrics,
			"memory", "", "", *config.MemoryTriggerThreshold, int(currentMetrics.MemoryUsage*100),
		); err != nil {
			return err
		}
	} else if shouldNotifyResourceResolved(config.MemoryTriggerThreshold, previousMetrics, currentMetrics, memoryUsage) {
		log.Info("sending memory alert resolved notification")
		if err := sendMetricNotification(
			ctx, true, deploymentTarget, *organization, config, previousMetrics, currentMetrics,
			"memory", "", "", *config.MemoryTriggerThreshold, int(currentMetrics.MemoryUsage*100),
		); err != nil {
			return err
		}
	}

	for _, diskMetric := range currentMetrics.DiskMetrics {
		var previousDiskMetric *types.DeploymentTargetDiskMetric
		if previousMetrics != nil {
			for _, m := range previousMetrics.DiskMetrics {
				if m.Device == diskMetric.Device && m.Path == diskMetric.Path {
					previousDiskMetric = &m
					break
				}
			}
		}

		if shouldNotifyResource(config.DiskTriggerThreshold, previousDiskMetric, diskMetric, diskUsage) {
			log.Info("sending disk alert notification", zap.String("device", diskMetric.Device), zap.String("path", diskMetric.Path))
			if err := sendMetricNotification(
				ctx, false, deploymentTarget, *organization, config, previousMetrics, currentMetrics,
				"disk", diskMetric.Device, diskMetric.Path, *config.DiskTriggerThreshold, int(diskMetric.Usage()*100),
			); err != nil {
				return err
			}
		} else if shouldNotifyResourceResolved(config.DiskTriggerThreshold, previousDiskMetric, diskMetric, diskUsage) {
			log.Info("sending disk alert resolved notification", zap.String("device", diskMetric.Device), zap.String("path", diskMetric.Path))
			if err := sendMetricNotification(
				ctx, true, deploymentTarget, *organization, config, previousMetrics, currentMetrics,
				"disk", diskMetric.Device, diskMetric.Path, *config.DiskTriggerThreshold, int(diskMetric.Usage()*100),
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func sendMetricNotification(
	ctx context.Context,
	resolved bool,
	deploymentTarget types.DeploymentTargetFull,
	organization types.Organization,
	config types.AlertConfiguration,
	previousMetrics *types.DeploymentTargetMetrics,
	currentMetrics types.DeploymentTargetMetrics,
	metricType string,
	diskDevice string,
	diskPath string,
	threshold int,
	usagePercent int,
) error {
	var aggErr error
	for _, user := range config.UserAccounts {
		var err error
		if resolved {
			err = mailsending.DeploymentTargetMetricsNotificationResolved(
				ctx, user, organization, deploymentTarget,
				metricType, diskDevice, diskPath, threshold, usagePercent,
			)
		} else {
			err = mailsending.DeploymentTargetMetricsNotificationAlert(
				ctx, user, organization, deploymentTarget,
				metricType, diskDevice, diskPath, threshold, usagePercent,
			)
		}
		if err != nil {
			internalctx.GetLogger(ctx).Warn("metric notification sending failed",
				zap.String("metricType", metricType),
				zap.Stringer("userId", user.ID),
				zap.Error(err),
			)
			aggErr = errors.Join(aggErr, err)
		}
	}

	record := types.NotificationRecord{
		OrganizationID:                   config.OrganizationID,
		CustomerOrganizationID:           config.CustomerOrganizationID,
		DeploymentTargetID:               &deploymentTarget.ID,
		AlertConfigurationID:             &config.ID,
		MetricType:                       &metricType,
		CurrentDeploymentTargetMetricsID: &currentMetrics.ID,
	}

	if diskDevice != "" {
		record.DiskDevice = &diskDevice
	}
	if diskPath != "" {
		record.DiskPath = &diskPath
	}
	if previousMetrics != nil {
		record.PreviousDeploymentTargetMetricsID = &previousMetrics.ID
	}
	if aggErr != nil {
		record.Message = aggErr.Error()
	}

	if err := db.SaveNotificationRecord(ctx, &record); err != nil {
		return fmt.Errorf("failed to save notification record: %w", err)
	}

	return nil
}

func shouldNotifyResource[T any](threshold *int, p *T, c T, f func(*T, T) (*float64, float64)) bool {
	pv, cv := f(p, c)
	return threshold != nil &&
		thresholdExceeded(*threshold, cv) &&
		(pv == nil || !thresholdExceeded(*threshold, *pv))
}

func shouldNotifyResourceResolved[T any](threshold *int, p *T, c T, f func(*T, T) (*float64, float64)) bool {
	pv, cv := f(p, c)
	return threshold != nil &&
		!thresholdExceeded(*threshold, cv) &&
		pv != nil && thresholdExceeded(*threshold, *pv)
}

func thresholdExceeded(threshold int, value float64) bool {
	return value*100 > float64(threshold)
}

func cpuUsage(p *types.DeploymentTargetMetrics, c types.DeploymentTargetMetrics) (*float64, float64) {
	return usageFunc(p, c, func(m types.DeploymentTargetMetrics) float64 { return m.CPUUsage })
}

func memoryUsage(p *types.DeploymentTargetMetrics, c types.DeploymentTargetMetrics) (*float64, float64) {
	return usageFunc(p, c, func(m types.DeploymentTargetMetrics) float64 { return m.MemoryUsage })
}

func diskUsage(p *types.DeploymentTargetDiskMetric, c types.DeploymentTargetDiskMetric) (*float64, float64) {
	return usageFunc(p, c, func(m types.DeploymentTargetDiskMetric) float64 { return m.Usage() })
}

func usageFunc[T any](p *T, c T, f func(T) float64) (*float64, float64) {
	if p != nil {
		return new(f(*p)), f(c)
	}
	return nil, f(c)
}
