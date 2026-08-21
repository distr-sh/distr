package main

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/google/uuid"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// deploymentMetricsNamespace is shared by the main loop with the metrics goroutines, which have
// no access to the resource polling themselves. Deployments are read from the agent's tracking
// secrets via GetExistingDeployments, like the logs watcher does.
var deploymentMetricsNamespace atomic.Pointer[string]

func watchDeploymentMetrics(ctx context.Context) {
	logger.Info("starting deployment metrics watch")
	tick := time.Tick(30 * time.Second)
	for ctx.Err() == nil {
		select {
		case <-tick:
			reportDeploymentMetrics(ctx)
		case <-ctx.Done():
			logger.Info("stopping to watch deployment metrics")
			return
		}
	}
}

type podUsage struct {
	cpuUsageMillis int64
	memoryBytes    int64
}

func reportDeploymentMetrics(ctx context.Context) {
	namespacePtr := deploymentMetricsNamespace.Load()
	if namespacePtr == nil {
		return
	}
	namespace := *namespacePtr

	deployments, err := GetExistingDeployments(ctx, namespace)
	if err != nil {
		logger.Error("could not get existing deployments for deployment metrics", zap.Error(err))
		return
	}
	if len(deployments) == 0 {
		return
	}

	pods, err := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error("listing pods for deployment metrics failed", zap.Error(err))
		return
	}

	podMetricses, err := metricsClientSet.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error("listing pod metrics failed", zap.Error(err))
		return
	}

	metricsByPod := make(map[string]*metricsv1beta1.PodMetrics, len(podMetricses.Items))
	for i := range podMetricses.Items {
		metricsByPod[podMetricses.Items[i].Name] = &podMetricses.Items[i]
	}

	resourceByPod := resolvePodResources(ctx, namespace, pods.Items)

	for _, deployment := range deployments {
		// Deployments created by very old agent versions may miss the ID (see isSameDeployment).
		if deployment.ID == uuid.Nil {
			continue
		}
		manifest, err := GetHelmManifest(ctx, namespace, deployment.ReleaseName)
		if err != nil {
			logger.Warn("could not get helm manifest for deployment metrics",
				zap.String("releaseName", deployment.ReleaseName), zap.Error(err))
			continue
		}

		manifestResources := make(map[string]struct{})
		for _, resource := range manifest {
			switch resource.GetKind() {
			case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet", "Job", "CronJob", "Pod":
				manifestResources[resource.GetKind()+"/"+resource.GetName()] = struct{}{}
			}
		}

		var resources []api.DeploymentResourceMetric
		for _, pod := range pods.Items {
			resource := resourceByPod[pod.Name]
			if _, ok := manifestResources[resource]; !ok {
				continue
			}
			podMetrics, ok := metricsByPod[pod.Name]
			if !ok {
				continue
			}
			for _, containerMetrics := range podMetrics.Containers {
				cpuLimitMillis, memoryLimitBytes := containerLimits(pod, containerMetrics.Name)
				resources = append(resources, api.DeploymentResourceMetric{
					Resource:         resource,
					Container:        containerName(pod, *podMetrics, containerMetrics),
					CPUUsageMillis:   containerMetrics.Usage.Cpu().MilliValue(),
					MemoryBytes:      containerMetrics.Usage.Memory().Value(),
					CPULimitMillis:   cpuLimitMillis,
					MemoryLimitBytes: memoryLimitBytes,
				})
			}
		}

		request := api.AgentDeploymentResourceMetricsRequest{Resources: resources}
		if err := agentClient.ReportDeploymentMetrics(ctx, deployment.ID, request); err != nil {
			logger.Error("failed to report deployment metrics",
				zap.String("releaseName", deployment.ReleaseName), zap.Error(err))
		}
	}
}

// containerName qualifies the pod name for multi-container pods. Whether the pod has more than
// one container is decided by the reported metrics, not by pod.Spec.Containers, because
// restartable init containers (sidecars) report usage but are not part of the spec list.
func containerName(
	pod corev1.Pod,
	podMetrics metricsv1beta1.PodMetrics,
	metrics metricsv1beta1.ContainerMetrics,
) string {
	if len(podMetrics.Containers) <= 1 {
		return pod.Name
	}

	return pod.Name + "/" + metrics.Name
}

// containerLimits returns the limits declared for one container of the pod. Init containers are
// searched too because restartable init containers (sidecars) report usage like regular ones.
func containerLimits(pod corev1.Pod, containerName string) (cpuLimitMillis, memoryLimitBytes *int64) {
	var container *corev1.Container
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == containerName {
			container = &pod.Spec.Containers[i]
			break
		}
	}
	if container == nil {
		for i := range pod.Spec.InitContainers {
			if pod.Spec.InitContainers[i].Name == containerName {
				container = &pod.Spec.InitContainers[i]
				break
			}
		}
	}
	if container == nil {
		return nil, nil
	}
	if quantity, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
		cpuLimitMillis = new(quantity.MilliValue())
	}
	if quantity, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
		memoryLimitBytes = new(quantity.Value())
	}
	return cpuLimitMillis, memoryLimitBytes
}

// resolvePodResources maps each pod to its resource key ("Kind/name") by resolving the
// controller owner chain: Pod -> ReplicaSet -> Deployment and Pod -> Job -> CronJob.
// Pods without a controller map to themselves ("Pod/name").
func resolvePodResources(ctx context.Context, namespace string, pods []corev1.Pod) map[string]string {
	replicaSetOwners := make(map[string]*metav1.OwnerReference)
	if replicaSets, err := k8sClient.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{}); err != nil {
		logger.Warn("listing replicasets for deployment metrics failed", zap.Error(err))
	} else {
		for _, replicaSet := range replicaSets.Items {
			replicaSetOwners[replicaSet.Name] = metav1.GetControllerOf(&replicaSet)
		}
	}

	jobOwners := make(map[string]*metav1.OwnerReference)
	if jobs, err := k8sClient.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{}); err != nil {
		logger.Warn("listing jobs for deployment metrics failed", zap.Error(err))
	} else {
		for _, job := range jobs.Items {
			jobOwners[job.Name] = metav1.GetControllerOf(&job)
		}
	}

	result := make(map[string]string, len(pods))
	for _, pod := range pods {
		owner := metav1.GetControllerOf(&pod)
		if owner == nil {
			result[pod.Name] = "Pod/" + pod.Name
			continue
		}
		switch owner.Kind {
		case "ReplicaSet":
			if parent := replicaSetOwners[owner.Name]; parent != nil && parent.Kind == "Deployment" {
				result[pod.Name] = "Deployment/" + parent.Name
			} else {
				result[pod.Name] = "ReplicaSet/" + owner.Name
			}
		case "Job":
			if parent := jobOwners[owner.Name]; parent != nil && parent.Kind == "CronJob" {
				result[pod.Name] = "CronJob/" + parent.Name
			} else {
				result[pod.Name] = "Job/" + owner.Name
			}
		default:
			result[pod.Name] = owner.Kind + "/" + owner.Name
		}
	}
	return result
}
