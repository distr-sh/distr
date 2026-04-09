package main

import (
	"fmt"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentauth"
	dockercommand "github.com/docker/cli/cli/command"
	dockerflags "github.com/docker/cli/cli/flags"
	composeapi "github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
)

func ComposeServiceForDeployment(
	deployment api.AgentDeployment,
	options ...compose.Option,
) (composeapi.Compose, error) {
	if cli, err := DockerCLIForDeployment(deployment); err != nil {
		return nil, err
	} else if svc, err := compose.NewComposeService(cli, options...); err != nil {
		return nil, fmt.Errorf("failed to create compose service: %w", err)
	} else {
		return svc, nil
	}
}

func DockerCLIForDeployment(deployment api.AgentDeployment) (dockercommand.Cli, error) {
	if dockerCli, err := dockercommand.NewDockerCli(); err != nil {
		return nil, fmt.Errorf("failed to create docker CLI: %w", err)
	} else if err = dockerCli.Initialize(DockerCLIOpts(deployment)); err != nil {
		return nil, fmt.Errorf("failed to initialize docker CLI: %w", err)
	} else {
		return dockerCli, err
	}
}

func DockerCLIOpts(deployment api.AgentDeployment) *dockerflags.ClientOptions {
	var opts dockerflags.ClientOptions
	if len(deployment.RegistryAuth) > 0 || hasRegistryImages(deployment) {
		opts.ConfigDir = agentauth.DockerConfigDir(deployment)
	}
	return &opts
}
