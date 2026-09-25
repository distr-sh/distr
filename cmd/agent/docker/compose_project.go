package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	composeapi "github.com/docker/compose/v5/pkg/api"
	"github.com/google/uuid"
)

const (
	composeFileName = "docker-compose.yaml"
	envFileName     = ".env"
)

// ComposeWorkingDir describes a directory that holds a deployment's compose file (and optional env file)
// under their canonical names, so that relative paths referenced from the compose file resolve.
type ComposeWorkingDir struct {
	// Path is the working directory containing the files.
	Path string
	// ComposeFile is the absolute path to the written compose file.
	ComposeFile string
	// EnvFile is the absolute path to the written env file, or empty if the deployment has no env file.
	EnvFile string
}

// ComposeProjectDir is the directory a deployment's compose file and env file are kept in for as long as
// the deployment exists. Running the lifecycle hooks of a project on teardown requires its compose file,
// and the deployment is gone from the resource response by then, so the files have to outlive the apply.
func ComposeProjectDir(id uuid.UUID) string {
	return filepath.Join(ScratchDir(), "projects", id.String())
}

func WriteComposeProjectDir(deployment api.AgentDeployment) (ComposeWorkingDir, error) {
	dir := ComposeProjectDir(deployment.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ComposeWorkingDir{}, fmt.Errorf("failed to create compose project directory: %w", err)
	}
	return writeComposeFiles(dir, deployment)
}

// EnsureComposeProjectDir writes the project files unless they already exist, which covers a deployment
// applied by an agent version that did not keep them yet and would otherwise never write them again.
func EnsureComposeProjectDir(deployment api.AgentDeployment) error {
	_, err := os.Stat(filepath.Join(ComposeProjectDir(deployment.ID), composeFileName))
	if err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_, err = WriteComposeProjectDir(deployment)
	return err
}

func ReadComposeProjectDir(id uuid.UUID) (ComposeWorkingDir, error) {
	dir := ComposeProjectDir(id)
	result := ComposeWorkingDir{Path: dir, ComposeFile: filepath.Join(dir, composeFileName)}
	if _, err := os.Stat(result.ComposeFile); err != nil {
		return ComposeWorkingDir{}, err
	}

	envFile := filepath.Join(dir, envFileName)
	if _, err := os.Stat(envFile); err == nil {
		result.EnvFile = envFile
	} else if !errors.Is(err, os.ErrNotExist) {
		return ComposeWorkingDir{}, err
	}

	return result, nil
}

func DeleteComposeProjectDir(id uuid.UUID) error {
	return os.RemoveAll(ComposeProjectDir(id))
}

// writeComposeFiles writes the compose file as "docker-compose.yaml" and, if present, the env file as
// ".env" into dir. Writing the env file as ".env" next to the compose file makes both project-level
// interpolation and service-level "env_file: - .env" directives resolve.
func writeComposeFiles(dir string, deployment api.AgentDeployment) (ComposeWorkingDir, error) {
	result := ComposeWorkingDir{Path: dir, ComposeFile: filepath.Join(dir, composeFileName)}
	if err := os.WriteFile(result.ComposeFile, deployment.ComposeFile, 0o600); err != nil {
		return ComposeWorkingDir{}, fmt.Errorf("failed to write compose file: %w", err)
	}

	envFile := filepath.Join(dir, envFileName)
	if deployment.EnvFile == nil {
		// Compose reads a ".env" left behind by a previous revision even when no env file is passed, so a
		// directory that is reused across revisions must not keep it.
		if err := os.Remove(envFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return ComposeWorkingDir{}, fmt.Errorf("failed to remove env file: %w", err)
		}
		return result, nil
	}

	if err := os.WriteFile(envFile, deployment.EnvFile, 0o600); err != nil {
		return ComposeWorkingDir{}, fmt.Errorf("failed to write env file: %w", err)
	}
	result.EnvFile = envFile

	return result, nil
}

func ComposeLoadOptions(workDir ComposeWorkingDir) composeapi.ProjectLoadOptions {
	opts := composeapi.ProjectLoadOptions{
		WorkingDir:  workDir.Path,
		ConfigPaths: []string{workDir.ComposeFile},
	}
	if workDir.EnvFile != "" {
		opts.EnvFiles = []string{workDir.EnvFile}
	}
	return opts
}

// LoadComposeProject loads the project of an applied deployment from its project directory. Compose runs
// the lifecycle hooks of a service only where it is given the service definition, which it cannot
// reconstruct from the container labels, so every operation that should run them has to pass the project.
// A Swarm deployment is rejected: it persists the name-stripped file that "docker stack deploy" consumes,
// so the project would end up named after the directory rather than after the deployment.
func LoadComposeProject(ctx context.Context, deployment AgentDeployment) (*composetypes.Project, error) {
	if deployment.DockerType != types.DockerTypeCompose {
		return nil, fmt.Errorf("cannot load the compose project of a %v deployment", deployment.DockerType)
	}

	workDir, err := ReadComposeProjectDir(deployment.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose project directory: %w", err)
	}
	project, err := composeService.LoadProject(ctx, ComposeLoadOptions(workDir))
	if err != nil {
		return nil, fmt.Errorf("failed to load compose project: %w", err)
	}
	return project, nil
}
