package main

import (
	"context"
	"time"

	"github.com/distr-sh/distr/api"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func watchMetrics(ctx context.Context) {
	logger.Info("starting metrics watch")
	tick := time.Tick(30 * time.Second)
	for ctx.Err() == nil {
		select {
		case <-tick:
			doReportMetrics(ctx)
		case <-ctx.Done():
			logger.Info("stopping to watch metrics")
			return
		}
	}
}

func doReportMetrics(ctx context.Context) {
	var cpuCapacityM int64
	var cpuUsageM int64
	var memoryCapacityBytes int64
	var memoryUsageBytes int64
	if nodes, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err != nil {
		logger.Error("getting nodes failed", zap.Error(err))
		return
	} else {
		for _, node := range nodes.Items {
			logger.Info("node", zap.String("name", node.Name))
			cpuCapacityM += node.Status.Capacity.Cpu().MilliValue()
			memoryCapacityBytes += node.Status.Capacity.Memory().Value()

			if nodeMetrics, err := metricsClientSet.MetricsV1beta1().NodeMetricses().
				Get(ctx, node.Name, metav1.GetOptions{}); err != nil {
				logger.Error("getting node metrics failed", zap.Error(err))
				return
			} else {
				logger.Debug("node metrics",
					zap.Any("node", node.Name),
					zap.Any("cpuUsage", nodeMetrics.Usage.Cpu().MilliValue()),
					zap.Any("memUsage", nodeMetrics.Usage.Memory().Value()))
				cpuUsageM += nodeMetrics.Usage.Cpu().MilliValue()
				memoryUsageBytes += nodeMetrics.Usage.Memory().Value()
			}
		}
	}

	logger.Debug("node metric sum", zap.Any("cpuUsageSum", cpuUsageM),
		zap.Any("memUsageSum", memoryUsageBytes))

	if cpuCapacityM > 0 && memoryCapacityBytes > 0 {
		if err := agentClient.ReportMetrics(ctx, api.AgentDeploymentTargetMetricsRequest{
			CPUCoresMillis: cpuCapacityM,
			CPUUsage:       float64(cpuUsageM) / float64(cpuCapacityM),
			MemoryBytes:    memoryCapacityBytes,
			MemoryUsage:    float64(memoryUsageBytes) / float64(memoryCapacityBytes),
		}); err != nil {
			logger.Error("failed to report metrics", zap.Error(err))
		}
	}
}
