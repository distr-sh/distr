package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/distr-sh/distr/api"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/opentofu/tofudl"
	"go.uber.org/zap"
)

var (
	defaultTofuPath string
	defaultTofuOnce sync.Once
	defaultTofuErr  error
)

func resolveTofuBinary(ctx context.Context, version string) (string, error) {
	if envPath := os.Getenv("DISTR_TOFU_PATH"); envPath != "" {
		logger.Info("using tofu binary from DISTR_TOFU_PATH", zap.String("path", envPath))
		return envPath, nil
	}

	if version != "" {
		return downloadTofuVersion(ctx, version)
	}

	defaultTofuOnce.Do(func() {
		if pathBin, err := exec.LookPath("tofu"); err == nil {
			logger.Info("found tofu binary in PATH", zap.String("path", pathBin))
			defaultTofuPath = pathBin
			return
		}

		defaultTofuPath, defaultTofuErr = downloadTofuVersion(ctx, "")
	})
	return defaultTofuPath, defaultTofuErr
}

func downloadTofuVersion(ctx context.Context, version string) (string, error) {
	logger.Info("downloading tofu via tofudl", zap.String("version", version))

	dl, err := tofudl.New()
	if err != nil {
		return "", fmt.Errorf("could not create tofudl downloader: %w", err)
	}

	var opts []tofudl.DownloadOpt
	if version != "" {
		opts = append(opts, tofudl.DownloadOptVersion(tofudl.Version(version)))
	}

	binary, err := dl.Download(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("could not download tofu binary: %w", err)
	}

	binName := "tofu"
	if version != "" {
		binName = fmt.Sprintf("tofu-%s", version)
	}

	binDir := filepath.Join(ScratchDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("could not create bin directory: %w", err)
	}

	binPath := filepath.Join(binDir, binName)
	if err := os.WriteFile(binPath, binary, 0o755); err != nil {
		return "", fmt.Errorf("could not write tofu binary: %w", err)
	}

	logger.Info("tofu binary downloaded", zap.String("path", binPath), zap.String("version", version))
	return binPath, nil
}

func hubBaseURL() string {
	loginEndpoint := os.Getenv("DISTR_LOGIN_ENDPOINT")
	return strings.TrimSuffix(loginEndpoint, "/api/v1/agent/login")
}

func tofuApply(ctx context.Context, deployment api.AgentDeployment) error {
	logger.Info("tofu apply",
		zap.String("deploymentId", deployment.ID.String()),
		zap.String("configUrl", deployment.TofuConfigURL),
		zap.String("configVersion", deployment.TofuConfigVersion))

	agentDeployment := NewAgentDeployment(deployment)
	agentDeployment.State = StateInstalling
	if err := SaveDeployment(*agentDeployment); err != nil {
		return fmt.Errorf("could not save deployment state: %w", err)
	}

	workspaceDir := WorkspaceDir(deployment.ID)

	if err := os.RemoveAll(workspaceDir); err != nil {
		agentDeployment.State = StateFailed
		_ = SaveDeployment(*agentDeployment)
		return fmt.Errorf("could not clean workspace before apply: %w", err)
	}

	if err := pullOCIArtifact(ctx, deployment, workspaceDir); err != nil {
		agentDeployment.State = StateFailed
		_ = SaveDeployment(*agentDeployment)
		return fmt.Errorf("could not pull OCI artifact: %w", err)
	}

	if err := writeVarsFile(workspaceDir, deployment.TofuVars); err != nil {
		agentDeployment.State = StateFailed
		_ = SaveDeployment(*agentDeployment)
		return fmt.Errorf("could not write tfvars file: %w", err)
	}

	tofuBin, err := resolveTofuBinary(ctx, deployment.TofuVersion)
	if err != nil {
		agentDeployment.State = StateFailed
		_ = SaveDeployment(*agentDeployment)
		return fmt.Errorf("could not resolve tofu binary: %w", err)
	}

	tf, err := tfexec.NewTerraform(workspaceDir, tofuBin)
	if err != nil {
		agentDeployment.State = StateFailed
		_ = SaveDeployment(*agentDeployment)
		return fmt.Errorf("could not create terraform executor: %w", err)
	}

	logWriter := &zapLogWriter{logger: logger, prefix: "tofu"}
	tf.SetStdout(logWriter)
	tf.SetStderr(logWriter)

	baseURL := hubBaseURL()
	targetID := os.Getenv("DISTR_TARGET_ID")
	targetSecret := os.Getenv("DISTR_TARGET_SECRET")
	stateURL := fmt.Sprintf("%s/api/v1/state/%s", baseURL, deployment.ID)

	lockURL := stateURL + "/lock"
	unlockURL := stateURL + "/unlock"

	initOpts := []tfexec.InitOption{
		tfexec.BackendConfig(fmt.Sprintf("address=%s", stateURL)),
		tfexec.BackendConfig(fmt.Sprintf("lock_address=%s", lockURL)),
		tfexec.BackendConfig(fmt.Sprintf("unlock_address=%s", unlockURL)),
		tfexec.BackendConfig("lock_method=POST"),
		tfexec.BackendConfig("unlock_method=POST"),
		tfexec.BackendConfig(fmt.Sprintf("username=%s", targetID)),
		tfexec.BackendConfig(fmt.Sprintf("password=%s", targetSecret)),
	}

	for key, value := range deployment.TofuBackendConfig {
		initOpts = append(initOpts, tfexec.BackendConfig(fmt.Sprintf("%s=%s", key, value)))
	}

	logger.Info("running tofu init", zap.String("deploymentId", deployment.ID.String()))
	if err := tf.Init(ctx, initOpts...); err != nil {
		agentDeployment.State = StateFailed
		_ = SaveDeployment(*agentDeployment)
		return fmt.Errorf("tofu init failed: %w", err)
	}

	planFile := filepath.Join(workspaceDir, "plan.out")
	logger.Info("running tofu plan", zap.String("deploymentId", deployment.ID.String()))
	hasChanges, err := tf.Plan(ctx, tfexec.Out(planFile))
	if err != nil {
		agentDeployment.State = StateFailed
		_ = SaveDeployment(*agentDeployment)
		return fmt.Errorf("tofu plan failed: %w", err)
	}

	if hasChanges {
		logger.Info("changes detected, running tofu apply", zap.String("deploymentId", deployment.ID.String()))
		if err := tf.Apply(ctx, tfexec.DirOrPlan(planFile)); err != nil {
			agentDeployment.State = StateFailed
			_ = SaveDeployment(*agentDeployment)
			return fmt.Errorf("tofu apply failed: %w", err)
		}
	} else {
		logger.Info("no changes detected", zap.String("deploymentId", deployment.ID.String()))
	}

	agentDeployment.State = StateInstalled
	if err := SaveDeployment(*agentDeployment); err != nil {
		return fmt.Errorf("could not save deployment state: %w", err)
	}

	return nil
}

func tofuDestroy(ctx context.Context, deployment AgentDeployment) error {
	logger.Info("tofu destroy",
		zap.String("deploymentId", deployment.ID.String()),
		zap.String("configUrl", deployment.TofuConfigURL),
		zap.String("configVersion", deployment.TofuConfigVersion))

	workspaceDir := WorkspaceDir(deployment.ID)

	tofuBin, err := resolveTofuBinary(ctx, deployment.TofuVersion)
	if err != nil {
		return fmt.Errorf("could not resolve tofu binary for destroy: %w", err)
	}

	tf, err := tfexec.NewTerraform(workspaceDir, tofuBin)
	if err != nil {
		return fmt.Errorf("could not create terraform executor for destroy: %w", err)
	}

	logWriter := &zapLogWriter{logger: logger, prefix: "tofu-destroy"}
	tf.SetStdout(logWriter)
	tf.SetStderr(logWriter)

	baseURL := hubBaseURL()
	targetID := os.Getenv("DISTR_TARGET_ID")
	targetSecret := os.Getenv("DISTR_TARGET_SECRET")
	stateURL := fmt.Sprintf("%s/api/v1/state/%s", baseURL, deployment.ID)

	initOpts := []tfexec.InitOption{
		tfexec.BackendConfig(fmt.Sprintf("address=%s", stateURL)),
		tfexec.BackendConfig(fmt.Sprintf("lock_address=%s/lock", stateURL)),
		tfexec.BackendConfig(fmt.Sprintf("unlock_address=%s/unlock", stateURL)),
		tfexec.BackendConfig("lock_method=POST"),
		tfexec.BackendConfig("unlock_method=POST"),
		tfexec.BackendConfig(fmt.Sprintf("username=%s", targetID)),
		tfexec.BackendConfig(fmt.Sprintf("password=%s", targetSecret)),
	}

	logger.Info("running tofu init for destroy", zap.String("deploymentId", deployment.ID.String()))
	if err := tf.Init(ctx, initOpts...); err != nil {
		return fmt.Errorf("tofu init for destroy failed: %w", err)
	}

	logger.Info("running tofu destroy", zap.String("deploymentId", deployment.ID.String()))
	if err := tf.Destroy(ctx); err != nil {
		return fmt.Errorf("tofu destroy failed: %w", err)
	}

	if err := os.RemoveAll(workspaceDir); err != nil {
		logger.Error("could not clean up workspace", zap.Error(err), zap.String("workspaceDir", workspaceDir))
	}

	return nil
}

func writeVarsFile(workspaceDir string, vars map[string]any) error {
	if len(vars) == 0 {
		return nil
	}

	var b strings.Builder
	for key, value := range vars {
		switch v := value.(type) {
		case string:
			b.WriteString(fmt.Sprintf("%s = %q\n", key, v))
		case bool:
			b.WriteString(fmt.Sprintf("%s = %t\n", key, v))
		case float64:
			if v == float64(int64(v)) {
				b.WriteString(fmt.Sprintf("%s = %d\n", key, int64(v)))
			} else {
				b.WriteString(fmt.Sprintf("%s = %g\n", key, v))
			}
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("could not marshal variable %q: %w", key, err)
			}
			b.WriteString(fmt.Sprintf("%s = %s\n", key, string(jsonBytes)))
		}
	}

	varsPath := filepath.Join(workspaceDir, "terraform.tfvars")
	return os.WriteFile(varsPath, []byte(b.String()), 0o644)
}
