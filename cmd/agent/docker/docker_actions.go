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
	dockerconfig "github.com/docker/cli/cli/config"
	composeapi "github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func DockerEngineApply(
	ctx context.Context,
	deployment api.AgentDeployment,
	updateStatus func(string),
) (agentDeployment *AgentDeployment, status string, err error) {
	logger := logger.With(zap.Stringer("deploymentId", deployment.ID))
	agentDeployment, err = NewAgentDeployment(deployment)
	if err != nil {
		return agentDeployment, status, err
	}

	agentDeployment.State = StateProgressing
	if err = SaveDeployment(*agentDeployment); err != nil {
		logger.Warn("failed to save deployment before apply", zap.Error(err))
	}

	if *deployment.DockerType == types.DockerTypeSwarm {
		logger.Debug("applying compose file in swarm mode")
		status, err = ApplyComposeFileSwarm(ctx, deployment, updateStatus)
	} else {
		logger.Debug("applying compose file")
		err = ApplyComposeFile(ctx, deployment, updateStatus)
		if err == nil {
			status = "compose command executed successfully"
		}
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

func ApplyComposeFile(ctx context.Context, deployment api.AgentDeployment, updateStatus func(string)) error {
	updateStatus("initializing compose service")

	var composeFileName string
	if f, err := WriteTempFile("distr-compose-*.yaml", deployment.ComposeFile); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	} else {
		composeFileName = string(f)
		defer f.Destroy()
	}

	var envFileName string
	if deployment.EnvFile != nil {
		if f, err := WriteTempFile("distr-*.env", deployment.EnvFile); err != nil {
			return fmt.Errorf("failed to write env file: %w", err)
		} else {
			envFileName = string(f)
			defer f.Destroy()
		}
	}

	eventProcessor := NewEventProcessor(updateStatus)
	composeService, err := ComposeServiceForDeployment(deployment, compose.WithEventProcessor(eventProcessor))
	if err != nil {
		return fmt.Errorf("failed to initialize compose service: %w", err)
	}

	loadOpts := composeapi.ProjectLoadOptions{ConfigPaths: []string{composeFileName}}
	if envFileName != "" {
		loadOpts.EnvFiles = []string{envFileName}
	}

	project, err := composeService.LoadProject(ctx, loadOpts)
	if err != nil {
		return fmt.Errorf("failed to load compose project: %w", err)
	}

	err = composeService.Up(ctx, project, composeapi.UpOptions{
		Create: composeapi.CreateOptions{RemoveOrphans: true},
		Start:  composeapi.StartOptions{Project: project},
	})
	if err != nil {
		return fmt.Errorf("compose up failed: %w", err)
	}

	return nil
}

func ApplyComposeFileSwarm(
	ctx context.Context,
	deployment api.AgentDeployment,
	updateStatus func(string),
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

	updateStatus("applying compose project")

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
