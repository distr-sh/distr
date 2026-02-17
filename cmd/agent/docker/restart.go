package main

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/compose/convert"
	composeapi "github.com/docker/compose/v5/pkg/api"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func RunDockerRestart(ctx context.Context, deployment AgentDeployment) error {
	switch deployment.DockerType {
	case types.DockerTypeCompose:
		return RunDockerComposeRestart(ctx, deployment)
	case types.DockerTypeSwarm:
		return RunDockerSwarmRestart(ctx, deployment)
	default:
		return fmt.Errorf("cannot restart deployment %v with type: %v", deployment.ProjectName, deployment.DockerType)
	}
}

func RunDockerComposeRestart(ctx context.Context, deployment AgentDeployment) error {
	err := composeService.Restart(ctx, deployment.ProjectName, composeapi.RestartOptions{})
	if err != nil {
		return fmt.Errorf("failed to restart deployment %v: %w", deployment.ProjectName, err)
	}
	return nil
}

func RunDockerSwarmRestart(ctx context.Context, deployment AgentDeployment) error {
	apiClient := dockerCli.Client()
	services, err := apiClient.ServiceList(
		ctx,
		swarm.ServiceListOptions{
			Filters: filters.NewArgs(filters.Arg("label", convert.LabelNamespace+"="+deployment.ProjectName)),
		},
	)
	if err != nil {
		return err
	}
	var aggErr error
	for _, svc := range services {
		var options swarm.ServiceUpdateOptions
		spec := svc.Spec
		spec.TaskTemplate.ForceUpdate++
		image := spec.TaskTemplate.ContainerSpec.Image
		if encodedAuth, err := command.RetrieveAuthTokenFromImage(dockerCli.ConfigFile(), image); err != nil {
			logger.Error("failed to retrieve encoded auth", zap.Error(err))
			multierr.AppendInto(&aggErr, err)
			continue
		} else {
			options.EncodedRegistryAuth = encodedAuth
		}
		response, err := apiClient.ServiceUpdate(
			ctx, svc.ID, svc.Version, spec,
			options,
		)
		if err != nil {
			logger.Error("failed to update service", zap.Error(err))
			multierr.AppendInto(&aggErr, err)
		}
		for _, w := range response.Warnings {
			logger.Warn("service update warning", zap.String("warning", w))
		}
	}
	return aggErr
}
