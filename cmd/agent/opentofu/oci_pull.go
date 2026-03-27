package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentauth"
	"github.com/distr-sh/distr/internal/agentenv"
	"go.uber.org/zap"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

const versionFile = ".distr-oci-version"

func pullOCIArtifact(ctx context.Context, deployment api.AgentDeployment, workspaceDir string) error {
	tag := deployment.TofuConfigVersion
	if tag == "" {
		tag = "latest"
	}

	expectedRef := fmt.Sprintf("%s:%s", deployment.TofuConfigURL, tag)
	if hasTFFiles(workspaceDir) && readPulledVersion(workspaceDir) == expectedRef {
		logger.Debug("workspace up to date, skipping OCI pull",
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

	repoRef := deployment.TofuConfigURL
	if agentenv.DistrRegistryHost != "" && !hasHostname(repoRef) {
		repoRef = agentenv.DistrRegistryHost + "/" + repoRef
	}
	repo, err := remote.NewRepository(repoRef)
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

	logger.Info("pulling OCI artifact",
		zap.String("url", deployment.TofuConfigURL),
		zap.String("tag", tag),
		zap.String("workspaceDir", workspaceDir))

	if _, err := oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions); err != nil {
		return fmt.Errorf("could not pull OCI artifact: %w", err)
	}

	_ = os.WriteFile(filepath.Join(workspaceDir, versionFile), []byte(expectedRef), 0o600)

	logger.Info("OCI artifact pulled successfully", zap.String("workspaceDir", workspaceDir))
	return nil
}

func readPulledVersion(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, versionFile))
	if err != nil {
		return ""
	}
	return string(data)
}

func hasHostname(ref string) bool {
	slashIdx := strings.Index(ref, "/")
	firstSegment := ref
	if slashIdx >= 0 {
		firstSegment = ref[:slashIdx]
	}
	return strings.Contains(firstSegment, ".") || strings.Contains(firstSegment, ":")
}

func hasTFFiles(dir string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tf"))
	if len(matches) > 0 {
		return true
	}
	matches, _ = filepath.Glob(filepath.Join(dir, "*.tf.json"))
	return len(matches) > 0
}
