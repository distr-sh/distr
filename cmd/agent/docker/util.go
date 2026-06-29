package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/distr-sh/distr/api"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func DecodeComposeFile(manifest []byte) (result map[string]any, err error) {
	err = yaml.Unmarshal(manifest, &result)
	return result, err
}

func EncodeComposeFile(compose map[string]any) (result []byte, err error) {
	return yaml.Marshal(compose)
}

// ComposeWorkingDir describes a temporary directory that holds a deployment's compose file (and optional
// env file) under their canonical names, so that relative paths referenced from the compose file resolve.
type ComposeWorkingDir struct {
	// Path is the working directory containing the files.
	Path string
	// ComposeFile is the absolute path to the written compose file.
	ComposeFile string
	// EnvFile is the absolute path to the written env file, or empty if the deployment has no env file.
	EnvFile string
}

// WriteComposeWorkingDir writes the deployment's compose file as "docker-compose.yaml" and, if present, its
// env file as ".env" into a fresh temporary directory. Writing the env file as ".env" next to the compose
// file makes both project-level interpolation and service-level "env_file: - .env" directives resolve. The
// returned cleanup function removes the directory and all its contents.
func WriteComposeWorkingDir(deployment api.AgentDeployment) (ComposeWorkingDir, func(), error) {
	dir, err := os.MkdirTemp("", "distr-compose-*")
	if err != nil {
		return ComposeWorkingDir{}, func() {}, fmt.Errorf("failed to create compose working directory: %w", err)
	}
	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			logger.Warn("failed to remove compose working directory", zap.String("dir", dir), zap.Error(err))
		}
	}

	result := ComposeWorkingDir{Path: dir, ComposeFile: filepath.Join(dir, "docker-compose.yaml")}
	if err := os.WriteFile(result.ComposeFile, deployment.ComposeFile, 0o600); err != nil {
		cleanup()
		return ComposeWorkingDir{}, func() {}, fmt.Errorf("failed to write compose file: %w", err)
	}

	if deployment.EnvFile != nil {
		result.EnvFile = filepath.Join(dir, ".env")
		if err := os.WriteFile(result.EnvFile, deployment.EnvFile, 0o600); err != nil {
			cleanup()
			return ComposeWorkingDir{}, func() {}, fmt.Errorf("failed to write env file: %w", err)
		}
	}

	return result, cleanup, nil
}
