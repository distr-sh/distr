package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/distr-sh/distr/internal/types"
	"github.com/docker/cli/cli/compose/convert"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
)

func CheckStatus(ctx context.Context, deployment AgentDeployment) (types.DeploymentStatusType, string, error) {
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
	deployment AgentDeployment,
) (types.DeploymentStatusType, string, error) {
	summaries, err := composeService.Ps(ctx, deployment.ProjectName, api.PsOptions{All: true})
	if err != nil {
		return types.DeploymentStatusTypeError, "", err
	}

	if len(summaries) == 0 {
		return types.DeploymentStatusTypeRunning, "deployment has no containers", nil
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
	deployment AgentDeployment,
) (types.DeploymentStatusType, string, error) {
	apiClient := dockerCli.Client()
	services, err := apiClient.ServiceList(
		ctx,
		swarm.ServiceListOptions{
			Filters: filters.NewArgs(filters.Arg("label", convert.LabelNamespace+"="+deployment.ProjectName)),
		},
	)
	if err != nil {
		return types.DeploymentStatusTypeError, "", err
	}
	for _, service := range services {
		if service.Spec.Mode.GlobalJob == nil && service.Spec.Mode.ReplicatedJob == nil {
			if service.ServiceStatus.RunningTasks < service.ServiceStatus.DesiredTasks {
				return types.DeploymentStatusTypeError, fmt.Sprintf("service %v is not running: running=%v, desired=%v",
					service.Spec.Name, service.ServiceStatus.RunningTasks, service.ServiceStatus.DesiredTasks), nil
			}
		}
	}
	return types.DeploymentStatusTypeHealthy, "status check passed", nil
}
