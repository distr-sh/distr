package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/controllerenv"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// watchStatus reports the status of the revision that was last applied successfully for every deployment,
// independently of the main loop, which may be busy applying or retrying a newer revision.
func watchStatus(ctx context.Context) {
	tick := time.Tick(controllerenv.Interval)
	for {
		select {
		case <-tick:
			reportStatus(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// statusReportMu is held by a status report from reading the deployment state until the report is sent, and by
// an install or upgrade while it saves the progressing state. Helm therefore cannot start changing a release
// while a report that saw no pending update is still checking it.
var statusReportMu sync.Mutex

// targetRevisionIDs maps every deployment in the last resource to its target revision. The main loop replaces
// the whole map and never modifies a published one.
var targetRevisionIDs atomic.Pointer[map[uuid.UUID]uuid.UUID]

func reportStatus(ctx context.Context) {
	namespace := controllerNamespace.Load()
	if namespace == nil {
		return
	}

	statusReportMu.Lock()
	defer statusReportMu.Unlock()

	deployments, err := GetExistingDeployments(ctx, *namespace)
	if err != nil {
		logger.Error("could not get existing deployments for status check", zap.Error(err))
		return
	}

	targets := targetRevisionIDs.Load()
	if targets == nil {
		return
	}

	for _, deployment := range deployments {
		revisionID := deployment.AppliedRevisionID()
		targetRevisionID, ok := (*targets)[deployment.ID]
		if revisionID == uuid.Nil || !ok {
			continue
		}

		statusType := types.DeploymentStatusTypeHealthy
		var message string
		if authErr := registryAuthErrors.Get(deployment.ID, revisionID); authErr != nil {
			statusType = types.DeploymentStatusTypeError
			message = authErr.Error()
		} else if message, err = CheckReleaseStatus(ctx, *namespace, deployment.ReleaseName); err != nil {
			logger.Warn("status check failed", zap.String("releaseName", deployment.ReleaseName), zap.Error(err))
			message = err.Error()
			// While an upgrade is being applied or retried, Helm is changing the applied revision's workloads.
			if requiresInstallOrUpgrade(targetRevisionID, &deployment) {
				statusType = types.DeploymentStatusTypeProgressing
			} else {
				statusType = types.DeploymentStatusTypeError
			}
		}
		current := api.ControllerDeployment{ID: deployment.ID, RevisionID: revisionID}
		if err := controllerClient.Status(ctx, current, statusType, message); err != nil {
			logger.Warn("status push failed", zap.Error(err))
		}
	}
}

var registryAuthErrors = &registryAuthErrorStore{errors: map[uuid.UUID]registryAuthError{}}

// registryAuthErrorStore holds the result of the last registry auth refresh of every deployment. The pull secret
// has to be refreshed while a revision is running, so a failure is reported for the applied revision as well.
type registryAuthErrorStore struct {
	mu     sync.Mutex
	errors map[uuid.UUID]registryAuthError
}

type registryAuthError struct {
	revisionID uuid.UUID
	err        error
}

func (s *registryAuthErrorStore) Set(deployment api.ControllerDeployment, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors[deployment.ID] = registryAuthError{revisionID: deployment.RevisionID, err: err}
}

func (s *registryAuthErrorStore) Get(deploymentID, revisionID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.errors[deploymentID]; ok && e.revisionID == revisionID {
		return e.err
	}
	return nil
}

func (s *registryAuthErrorStore) Delete(deploymentID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.errors, deploymentID)
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
