package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/compose-spec/compose-go/v2/dotenv"
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentauth"
	"github.com/distr-sh/distr/internal/agentenv"
	"github.com/distr-sh/distr/internal/types"
	dockercommand "github.com/docker/cli/cli/command"
	dockerconfig "github.com/docker/cli/cli/config"
	dockerflags "github.com/docker/cli/cli/flags"
	composeapi "github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func DockerEngineApply(
	ctx context.Context,
	deployment api.AgentDeployment,
	statusCh chan<- string,
) (agentDeployment *AgentDeployment, status string, err error) {
	defer close(statusCh)

	agentDeployment, err = NewAgentDeployment(deployment)
	if err != nil {
		return agentDeployment, status, err
	}

	agentDeployment.State = StateProgressing
	if err = SaveDeployment(*agentDeployment); err != nil {
		logger.Warn("failed to save deployment before apply", zap.Error(err))
	}

	if *deployment.DockerType == types.DockerTypeSwarm {
		status, err = ApplyComposeFileSwarm(ctx, deployment, statusCh)
	} else {
		status, err = ApplyComposeFile(ctx, deployment, statusCh)
	}

	if err == nil {
		agentDeployment.State = StateReady
	} else {
		agentDeployment.State = StateFailed
	}

	if err1 := SaveDeployment(*agentDeployment); err1 != nil {
		logger.Warn("failed to save deployment after apply", zap.Error(err1))
	}

	return agentDeployment, status, err
}

func ApplyComposeFile(
	ctx context.Context,
	deployment api.AgentDeployment,
	statusCh chan<- string,
) (string, error) {
	var envFile *os.File
	var err error

	if deployment.EnvFile != nil {
		if envFile, err = os.CreateTemp("", "distr-env"); err != nil {
			logger.Error("", zap.Error(err))
			return "", fmt.Errorf("failed to create env file in tmp directory: %w", err)
		} else {
			if _, err = envFile.Write(deployment.EnvFile); err != nil {
				logger.Error("", zap.Error(err))
				return "", fmt.Errorf("failed to write env file: %w", err)
			}
			_ = envFile.Close()
			defer func() {
				if err := os.Remove(envFile.Name()); err != nil {
					logger.Error("failed to remove env file from tmp directory", zap.Error(err))
				}
			}()
		}
	}

	statusCh <- "pulling docker images"
	if err := PullComposeImages(ctx, deployment, statusCh); err != nil {
		return "", fmt.Errorf("failed to pull compose images: %w", err)
	}

	statusCh <- "applying compose project"
	composeArgs := []string{"compose"}
	if envFile != nil {
		composeArgs = append(composeArgs, fmt.Sprintf("--env-file=%v", envFile.Name()))
	}
	composeArgs = append(composeArgs, "-f", "-", "up", "-d", "--quiet-pull", "--remove-orphans")

	cmd := exec.CommandContext(ctx, "docker", composeArgs...)
	cmd.Stdin = bytes.NewReader(deployment.ComposeFile)
	cmd.Env = append(os.Environ(), DockerConfigEnv(deployment)...)

	var cmdOut []byte
	cmdOut, err = cmd.CombinedOutput()
	statusStr := string(cmdOut)
	logger.Debug("docker compose returned", zap.String("output", statusStr), zap.Error(err))

	if err != nil {
		return "", errors.New(statusStr)
	} else {
		return statusStr, nil
	}
}

type pullEventProcessor struct {
	statusCh chan<- string
}

func (p *pullEventProcessor) Start(_ context.Context, _ string) {}
func (p *pullEventProcessor) Done(_ string, _ bool)             {}
func (p *pullEventProcessor) On(events ...composeapi.Resource) {
	for _, e := range events {
		var msg string
		switch e.Text {
		case composeapi.StatusPulling, composeapi.StatusPulled:
			// e.ID is "Image nginx:latest" for image-level events
			image := strings.TrimPrefix(e.ID, "Image ")
			msg = fmt.Sprintf("%s: %s", image, e.Text)
		case composeapi.StatusDownloading, composeapi.StatusDownloadComplete:
			// e.ParentID is the image name, e.ID is the layer digest
			image := strings.TrimPrefix(e.ParentID, "Image ")
			msg = fmt.Sprintf("%s (%s): %s", image, e.ID, e.Text)
			if e.Percent > 0 {
				msg = fmt.Sprintf("%s (%d%%)", msg, e.Percent)
			}
		default:
			continue
		}
		select {
		case p.statusCh <- msg:
		default:
		}
	}
}

func PullComposeImages(ctx context.Context, deployment api.AgentDeployment, statusCh chan<- string) error {
	composeFile, err := os.CreateTemp("", "distr-compose-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp compose file: %w", err)
	}
	defer func() {
		if err := os.Remove(composeFile.Name()); err != nil {
			logger.Error("failed to remove temp compose file", zap.Error(err))
		}
	}()
	if _, err := composeFile.Write(deployment.ComposeFile); err != nil {
		return fmt.Errorf("failed to write temp compose file: %w", err)
	}
	_ = composeFile.Close()

	dockerCli, err := dockercommand.NewDockerCli()
	if err != nil {
		return fmt.Errorf("failed to create docker CLI object: %w", err)
	}

	if err := dockerCli.Initialize(DockerCLIOpts(deployment)); err != nil {
		return fmt.Errorf("failed to initialize docker CLI object: %w", err)
	}

	composeService, err := compose.NewComposeService(dockerCli, compose.WithEventProcessor(&pullEventProcessor{statusCh: statusCh}))
	if err != nil {
		return fmt.Errorf("failed to initialize compose service: %w", err)
	}

	project, err := composeService.LoadProject(ctx, composeapi.ProjectLoadOptions{
		ConfigPaths: []string{composeFile.Name()},
	})
	if err != nil {
		return fmt.Errorf("failed to load compose project: %w", err)
	}

	if err = composeService.Pull(ctx, project, composeapi.PullOptions{}); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	return nil
}

func ApplyComposeFileSwarm(
	ctx context.Context,
	deployment api.AgentDeployment,
	statusCh chan<- string,
) (string, error) {
	// Step 1 Ensure Docker Swarm is initialized
	initCmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.Swarm.LocalNodeState}}")
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to check Docker Swarm state", zap.Error(err))
		return "", fmt.Errorf("failed to check Docker Swarm state: %w", err)
	}

	if !strings.Contains(strings.TrimSpace(string(initOutput)), "active") {
		logger.Error("Docker Swarm not initialized", zap.String("output", string(initOutput)))
		return "", fmt.Errorf("docker Swarm not initialized: %s", string(initOutput))
	}

	projectName, err := getProjectName(deployment.ComposeFile)
	if err != nil {
		return "", fmt.Errorf("failed to get project name from compose file: %w", err)
	}

	cleanedComposeFile, err := cleanComposeFile(deployment.ComposeFile)
	if err != nil {
		return "", err
	}

	// Construct environment variables
	envVars := os.Environ()
	envVars = append(envVars, DockerConfigEnv(deployment)...)

	// // If an env file is provided, load its values
	if deployment.EnvFile != nil {
		parsedEnv, err := dotenv.UnmarshalBytesWithLookup(deployment.EnvFile, nil)
		if err != nil {
			return "", fmt.Errorf("failed to parse env file: %w", err)
		}
		for key, value := range parsedEnv {
			envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
		}
	}

	statusCh <- "applying compose project"

	// Deploy the stack
	composeArgs := []string{
		"stack", "deploy",
		"--compose-file", "-",
		"--with-registry-auth",
		"--detach=true",
		projectName,
	}
	cmd := exec.CommandContext(ctx, "docker", composeArgs...)
	cmd.Stdin = bytes.NewReader(cleanedComposeFile)
	cmd.Env = envVars // Ensure the same env variables are used

	// Execute the command and capture output
	cmdOut, err := cmd.CombinedOutput()
	statusStr := string(cmdOut)

	if err != nil {
		logger.Error("docker stack deploy failed", zap.String("output", statusStr))
		return "", errors.New(statusStr)
	} else {
		logger.Debug("docker stack deploy returned", zap.String("output", statusStr), zap.Error(err))
	}

	return statusStr, nil
}

func DockerEngineUninstall(ctx context.Context, deployment AgentDeployment) error {
	if deployment.DockerType == types.DockerTypeSwarm {
		return UninstallDockerSwarm(ctx, deployment)
	}
	return UninstallDockerCompose(ctx, deployment)
}

func UninstallDockerCompose(ctx context.Context, deployment AgentDeployment) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "--project-name", deployment.ProjectName, "down", "--volumes")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %v", err, string(out))
	}
	return nil
}

func UninstallDockerSwarm(ctx context.Context, deployment AgentDeployment) error {
	cmd := exec.CommandContext(ctx, "docker", "stack", "rm", deployment.ProjectName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove Docker Swarm stack: %w: %v", err, string(out))
	}

	// Optional: Prune unused networks created by Swarm
	pruneCmd := exec.CommandContext(ctx, "docker", "network", "prune", "-f")
	pruneOut, pruneErr := pruneCmd.CombinedOutput()
	if pruneErr != nil {
		logger.Warn("Failed to prune networks", zap.String("output", string(pruneOut)), zap.Error(pruneErr))
	}

	return nil
}

func cleanComposeFile(composeData []byte) ([]byte, error) {
	if compose, err := DecodeComposeFile(composeData); err != nil {
		return nil, err
	} else {
		delete(compose, "name")
		return EncodeComposeFile(compose)
	}
}

func DockerConfigEnv(deployment api.AgentDeployment) []string {
	if len(deployment.RegistryAuth) > 0 || hasRegistryImages(deployment) {
		return []string{
			dockerconfig.EnvOverrideConfigDir + "=" + agentauth.DockerConfigDir(deployment),
		}
	} else {
		return nil
	}
}

func DockerCLIOpts(deployment api.AgentDeployment) *dockerflags.ClientOptions {
	var opts dockerflags.ClientOptions
	if len(deployment.RegistryAuth) > 0 || hasRegistryImages(deployment) {
		opts.ConfigDir = agentauth.DockerConfigDir(deployment)
	}
	return &opts
}

// hasRegistryImages parses the compose file in order to check whether one of the services uses an image hosted on
// [agentenv.DistrRegistryHost].
func hasRegistryImages(deployment api.AgentDeployment) bool {
	var compose struct {
		Services map[string]struct {
			Image string
		}
	}
	if err := yaml.Unmarshal(deployment.ComposeFile, &compose); err != nil {
		return false
	}
	for _, svc := range compose.Services {
		if strings.HasPrefix(svc.Image, agentenv.DistrRegistryHost) {
			return true
		}
	}
	return false
}
