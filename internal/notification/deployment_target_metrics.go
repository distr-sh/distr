package notification

import (
	"context"
	"fmt"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/types"
	"go.uber.org/zap"
)

func SendDeploymentTargetMetricsNotifications(
	ctx context.Context,
	deploymentTarget types.DeploymentTarget,
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
			return fmt.Errorf("failed to send deployment status notifications with config: %w", err)
		}
	}

	return nil
}

func sendDeploymentTargetMetricsNotificationsWithConfig(
	ctx context.Context,
	deploymentTarget types.DeploymentTarget,
	previousMetrics *types.DeploymentTargetMetrics,
	currentMetrics types.DeploymentTargetMetrics,
	config types.AlertConfiguration,
) error {
	if !config.Enabled || !config.AnyThresholdEnabled() {
		return nil
	}

	log := internalctx.GetLogger(ctx).With(zap.Stringer("configId", config.ID))

	if shouldNotifyResource(config.CpuTriggerThreshold, previousMetrics, currentMetrics, cpuUsage) {
		log.Debug("TODO: send CPU alert")
	} else if shouldNotifyResourceResolved(config.CpuTriggerThreshold, previousMetrics, currentMetrics, cpuUsage) {
		log.Debug("TODO: send CPU alert resolved")
	}

	if shouldNotifyResource(config.MemoryTriggerThreshold, previousMetrics, currentMetrics, memoryUsage) {
		log.Debug("TODO: send memory alert")
	} else if shouldNotifyResourceResolved(config.MemoryTriggerThreshold, previousMetrics, currentMetrics, memoryUsage) {
		log.Debug("TODO: send memory alert resolved")
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
			log.Debug("TODO: send disk alert")
		} else if shouldNotifyResourceResolved(config.DiskTriggerThreshold, previousDiskMetric, diskMetric, diskUsage) {
			log.Debug("TODO: send disk alert resolved")
		}
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
	return int(value*100) > threshold
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
