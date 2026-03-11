package main

import (
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

func ScratchDir() string {
	if dir := os.Getenv("DISTR_AGENT_SCRATCH_DIR"); dir != "" {
		return dir
	}
	return "./scratch"
}

func WorkspaceDir(deploymentID uuid.UUID) string {
	return filepath.Join(ScratchDir(), "ws", deploymentID.String())
}

func DeploymentsDir() string {
	return filepath.Join(ScratchDir(), "deployments")
}
