package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/controllerenv"
	"github.com/distr-sh/distr/internal/types"
	"github.com/docker/cli/cli/compose/convert"
	composeapi "github.com/docker/compose/v5/pkg/api"
	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	mobyClient "github.com/moby/moby/client"
	"go.uber.org/zap"
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
// an apply while it saves the progressing state. An apply therefore cannot start replacing containers while a
// report that saw no pending update is still checking them.
var statusReportMu sync.Mutex

// targetRevisionIDs maps every deployment in the last resource to its target revision. The main loop replaces
// the whole map and never modifies a published one.
var targetRevisionIDs atomic.Pointer[map[uuid.UUID]uuid.UUID]

func reportStatus(ctx context.Context) {
	statusReportMu.Lock()
	defer statusReportMu.Unlock()

	deployments, err := GetExistingDeployments()
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

		statusType, message, err := CheckStatus(ctx, deployment)
		if err != nil {
			statusType, message = types.DeploymentStatusTypeError, err.Error()
		}
		// While an update is being applied or retried, the applied revision's containers are being replaced.
		if statusType == types.DeploymentStatusTypeError && requiresApply(targetRevisionID, &deployment) {
			statusType = types.DeploymentStatusTypeProgressing
		}

		current := api.ControllerDeployment{ID: deployment.ID, RevisionID: revisionID}
		if err := client.Status(ctx, current, statusType, message); err != nil {
			logger.Warn("failed to send status", zap.Error(err))
		}
	}
}

func CheckStatus(ctx context.Context, deployment ControllerDeployment) (types.DeploymentStatusType, string, error) {
	switch deployment.DockerType {
	case types.DockerTypeCompose:
		return CheckDockerComposeStatus(ctx, deployment)
	case types.DockerTypeSwarm:
		return CheckDockerSwarmStatus(ctx, deployment)
	default:
		return types.DeploymentStatusTypeError, "", fmt.Errorf("unknown docker type: %v", deployment.DockerType)
	}
}

func CheckDockerComposeStatus(
	ctx context.Context,
	deployment ControllerDeployment,
) (types.DeploymentStatusType, string, error) {
	summaries, err := composeService.Ps(ctx, deployment.ProjectName, composeapi.PsOptions{All: true})
	if err != nil {
		return types.DeploymentStatusTypeError, "", err
	}

	if len(summaries) == 0 {
		return types.DeploymentStatusTypeError, "deployment has no containers", nil
	}

	var healthyCount, runningCount, startingCount int
	for _, summary := range summaries {
		switch summary.State {
		case container.StateRestarting:
			startingCount++
		case container.StateRunning:
			switch summary.Health {
			case container.Healthy:
				healthyCount++
			case container.Starting:
				startingCount++
			case container.NoHealthcheck, "":
				runningCount++
			default:
				return types.DeploymentStatusTypeError,
					fmt.Sprintf("service %v is not healthy: state=%v, health=%v, status=%v, exitCode=%v",
						summary.Name, summary.State, summary.Health, summary.Status, summary.ExitCode),
					nil
			}
		default:
			return types.DeploymentStatusTypeError,
				fmt.Sprintf("service %v is not in running state: state=%v, status=%v, exitCode=%v",
					summary.Name, summary.State, summary.Status, summary.ExitCode),
				nil
		}
	}
	var msgParts []string
	if healthyCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d healthy", healthyCount))
	}
	if runningCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d running (healthchecks missing)", runningCount))
	}
	if startingCount > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%d starting", startingCount))
	}
	msg := "status check results: " + strings.Join(msgParts, ", ")

	if startingCount > 0 {
		return types.DeploymentStatusTypeProgressing, msg, nil
	} else if runningCount > 0 {
		return types.DeploymentStatusTypeRunning, msg, nil
	} else {
		return types.DeploymentStatusTypeHealthy, msg, nil
	}
}

func CheckDockerSwarmStatus(
	ctx context.Context,
	deployment ControllerDeployment,
) (types.DeploymentStatusType, string, error) {
	apiClient := dockerCli.Client()
	services, err := apiClient.ServiceList(
		ctx,
		mobyClient.ServiceListOptions{
			Filters: mobyClient.Filters{}.Add("label", convert.LabelNamespace+"="+deployment.ProjectName),
			Status:  true,
		},
	)
	if err != nil {
		return types.DeploymentStatusTypeError, "", err
	}
	if len(services.Items) == 0 {
		return types.DeploymentStatusTypeError, "deployment has no services", nil
	}
	for _, service := range services.Items {
		if service.Spec.Mode.GlobalJob == nil && service.Spec.Mode.ReplicatedJob == nil {
			if service.ServiceStatus == nil {
				return types.DeploymentStatusTypeError, fmt.Sprintf("service %v has nil ServiceStatus", service.Spec.Name), nil
			}
			if service.ServiceStatus.RunningTasks < service.ServiceStatus.DesiredTasks {
				return types.DeploymentStatusTypeError, fmt.Sprintf("service %v is not running: running=%v, desired=%v",
					service.Spec.Name, service.ServiceStatus.RunningTasks, service.ServiceStatus.DesiredTasks), nil
			}
		}
	}
	return types.DeploymentStatusTypeHealthy, "status check passed", nil
}
