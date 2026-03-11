package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentauth"
	"github.com/distr-sh/distr/internal/agentenv"
	"go.uber.org/zap"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

func pullOCIArtifact(ctx context.Context, deployment api.AgentDeployment, workspaceDir string) error {
	if hasTFFiles(workspaceDir) {
		logger.Info("workspace already has .tf files, skipping OCI pull",
			zap.String("workspaceDir", workspaceDir))
		return nil
	}

	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return fmt.Errorf("could not create workspace directory: %w", err)
	}

	authClient, err := agentauth.EnsureAuth(ctx, client.RawToken(), deployment)
	if err != nil {
		return fmt.Errorf("could not authenticate with registry: %w", err)
	}

	repo, err := remote.NewRepository(deployment.TofuConfigURL)
	if err != nil {
		return fmt.Errorf("could not create repository reference: %w", err)
	}
	repo.PlainHTTP = agentenv.DistrRegistryPlainHTTP
	repo.Client = authClient

	fs, err := file.New(workspaceDir)
	if err != nil {
		return fmt.Errorf("could not create file store: %w", err)
	}
	defer fs.Close()

	tag := deployment.TofuConfigVersion
	if tag == "" {
		tag = "latest"
	}

	logger.Info("pulling OCI artifact",
		zap.String("url", deployment.TofuConfigURL),
		zap.String("tag", tag),
		zap.String("workspaceDir", workspaceDir))

	if _, err := oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions); err != nil {
		return fmt.Errorf("could not pull OCI artifact: %w", err)
	}

	logger.Info("OCI artifact pulled successfully", zap.String("workspaceDir", workspaceDir))
	return nil
}

func hasTFFiles(dir string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, "*.tf"))
	if err != nil {
		return false
	}
	return len(matches) > 0
}
