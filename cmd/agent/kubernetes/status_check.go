package main

import (
	"context"
	"fmt"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentenv"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// watchStatus reports the status of the revision that was last applied successfully for every deployment,
// independently of the main loop, which may be busy applying a newer revision.
func watchStatus(ctx context.Context) {
	tick := time.Tick(agentenv.Interval)
	for {
		select {
		case <-tick:
			reportStatus(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func reportStatus(ctx context.Context) {
	namespace := agentNamespace.Load()
	if namespace == nil {
		return
	}

	deployments, err := GetExistingDeployments(ctx, *namespace)
	if err != nil {
		logger.Error("could not get existing deployments for status check", zap.Error(err))
		return
	}

	for _, deployment := range deployments {
		revisionID := deployment.AppliedRevisionID()
		if revisionID == uuid.Nil || deployment.State == StateProgressing {
			continue
		}

		statusType := types.DeploymentStatusTypeHealthy
		message, err := CheckReleaseStatus(ctx, *namespace, deployment.ReleaseName)
		if err != nil {
			logger.Warn("status check failed", zap.String("releaseName", deployment.ReleaseName), zap.Error(err))
			statusType = types.DeploymentStatusTypeError
			message = err.Error()
		}
		current := api.AgentDeployment{ID: deployment.ID, RevisionID: revisionID}
		if err := agentClient.Status(ctx, current, statusType, message); err != nil {
			logger.Warn("status push failed", zap.Error(err))
		}
	}
}

func CheckReleaseStatus(ctx context.Context, namespace, releaseName string) (string, error) {
	resources, err := GetHelmManifest(ctx, namespace, releaseName)
	if err != nil {
		return "", fmt.Errorf("could not get helm manifest: %w", err)
	}
	for _, resource := range resources {
		logger.Sugar().Debugf("check status for %v %v", resource.GetKind(), resource.GetName())
		if err := CheckStatus(ctx, namespace, resource); err != nil {
			return "", fmt.Errorf("resource status error: %w", err)
		}
	}
	return fmt.Sprintf("status check passed. %v resources healthy", len(resources)), nil
}

func CheckStatus(ctx context.Context, namespace string, resource *unstructured.Unstructured) error {
	switch resource.GetKind() {
	case "Deployment":
		if deployment, err := k8sClient.AppsV1().Deployments(namespace).
			Get(ctx, resource.GetName(), metav1.GetOptions{}); err != nil {
			return err
		} else if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
			return ReplicasError(resource, deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)
		}
	case "StatefulSet":
		if statefulSet, err := k8sClient.AppsV1().StatefulSets(namespace).
			Get(ctx, resource.GetName(), metav1.GetOptions{}); err != nil {
			return err
		} else if statefulSet.Status.ReadyReplicas < *statefulSet.Spec.Replicas {
			return ReplicasError(resource, statefulSet.Status.ReadyReplicas, *statefulSet.Spec.Replicas)
		}
	case "DaemonSet":
		if daemonSet, err := k8sClient.AppsV1().DaemonSets(namespace).
			Get(ctx, resource.GetName(), metav1.GetOptions{}); err != nil {
			return err
		} else if daemonSet.Status.NumberUnavailable > 0 {
			return ReplicasError(resource, daemonSet.Status.NumberReady, daemonSet.Status.DesiredNumberScheduled)
		}
	}
	return nil
}

func ReplicasError(resource *unstructured.Unstructured, ready, desired int32) error {
	return ResourceStatusError(resource, fmt.Sprintf("ReadyReplicas (%v) is less than desired (%v)", ready, desired))
}

func ResourceStatusError(resource *unstructured.Unstructured, msg string) error {
	return fmt.Errorf("%v %v status check failed: %v", resource.GetKind(), resource.GetName(), msg)
}
