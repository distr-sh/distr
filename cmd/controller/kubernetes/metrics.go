package main

import (
	"context"
	"errors"
	"os"
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
		reportMetrics := api.ControllerDeploymentTargetMetricsRequest{
			CPUCoresMillis: cpuCapacityM,
			CPUUsage:       float64(cpuUsageM) / float64(cpuCapacityM),
			MemoryBytes:    memoryCapacityBytes,
			MemoryUsage:    float64(memoryUsageBytes) / float64(memoryCapacityBytes),
		}

		if usage, err := controllerSelfUsage(ctx); err != nil {
			logger.Warn("failed to collect controller self metrics", zap.Error(err))
		} else {
			reportMetrics.ControllerCPUUsageMillis = &usage.cpuUsageMillis
			reportMetrics.ControllerMemoryBytes = &usage.memoryBytes
		}

		if err := controllerClient.ReportMetrics(ctx, reportMetrics); err != nil {
			logger.Error("failed to report metrics", zap.Error(err))
		}
	}
}

// controllerSelfUsage returns the usage of the controller's own pod. The pod name is the hostname
// (the controller pod does not override it), the namespace comes from the state shared by the
// main loop, which is always stored before the metrics goroutine is started.
func controllerSelfUsage(ctx context.Context) (*podUsage, error) {
	namespace := deploymentMetricsNamespace.Load()
	if namespace == nil {
		return nil, errors.New("namespace is not known yet")
	}
	podName, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	podMetrics, err := metricsClientSet.MetricsV1beta1().PodMetricses(*namespace).
		Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var usage podUsage
	for _, container := range podMetrics.Containers {
		usage.cpuUsageMillis += container.Usage.Cpu().MilliValue()
		usage.memoryBytes += container.Usage.Memory().Value()
	}
	return &usage, nil
}
