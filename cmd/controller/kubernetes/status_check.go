package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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
