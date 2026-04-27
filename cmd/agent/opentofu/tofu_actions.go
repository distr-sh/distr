package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/distr-sh/distr/api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/opentofu/tofudl"
	"go.uber.org/zap"
)

var (
	validHCLIdentifier  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	reservedBackendKeys = map[string]bool{
		"address":        true,
		"lock_address":   true,
		"unlock_address": true,
		"lock_method":    true,
		"unlock_method":  true,
		"username":       true,
		"password":       true,
	}
)

var (
	defaultTofuPath string
	defaultTofuMu   sync.Mutex

	hubBase      string
	targetID     string
	targetSecret string
)

func initHubConfig() {
	hubBase = strings.TrimSuffix(os.Getenv("DISTR_LOGIN_ENDPOINT"), "/api/v1/agent/login")
	targetID = os.Getenv("DISTR_TARGET_ID")
	targetSecret = os.Getenv("DISTR_TARGET_SECRET")
}

func resolveTofuBinary() (string, error) {
	if envPath := os.Getenv("DISTR_TOFU_PATH"); envPath != "" {
		logger.Info("using tofu binary from DISTR_TOFU_PATH", zap.String("path", envPath))
		return envPath, nil
	}

	defaultTofuMu.Lock()
	defer defaultTofuMu.Unlock()

	if defaultTofuPath != "" {
		return defaultTofuPath, nil
	}

	if pathBin, err := exec.LookPath("tofu"); err == nil {
		logger.Info("found tofu binary in PATH", zap.String("path", pathBin))
		defaultTofuPath = pathBin
		return defaultTofuPath, nil
	}

	path, err := downloadDefaultTofu()
	if err != nil {
		return "", err
	}
	defaultTofuPath = path
	return defaultTofuPath, nil
}

func downloadDefaultTofu() (string, error) {
	logger.Info("downloading tofu via tofudl")

	dl, err := tofudl.New()
	if err != nil {
		return "", fmt.Errorf("could not create tofudl downloader: %w", err)
	}

	binary, err := dl.Download(context.Background())
	if err != nil {
		return "", fmt.Errorf("could not download tofu binary: %w", err)
	}

	binDir := filepath.Join(ScratchDir(), "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("could not create bin directory: %w", err)
	}

	binPath := filepath.Join(binDir, "tofu")
	if err := os.WriteFile(binPath, binary, 0o755); err != nil {
		return "", fmt.Errorf("could not write tofu binary: %w", err)
	}

	logger.Info("tofu binary downloaded", zap.String("path", binPath))
	return binPath, nil
}

type tofuExecutor struct {
	tf           *tfexec.Terraform
	workspaceDir string
	initOpts     []tfexec.InitOption
}

func prepareTofuExecutor(
	deploymentID uuid.UUID,
	backendConfig map[string]string, logPrefix string,
) (*tofuExecutor, error) {
	workspaceDir := WorkspaceDir(deploymentID)

	tofuBin, err := resolveTofuBinary()
	if err != nil {
		return nil, fmt.Errorf("could not resolve tofu binary: %w", err)
	}

	tf, err := tfexec.NewTerraform(workspaceDir, tofuBin)
	if err != nil {
		return nil, fmt.Errorf("could not create terraform executor: %w", err)
	}

	logWriter := &zapLogWriter{logger: logger, prefix: logPrefix}
	tf.SetStdout(logWriter)
	tf.SetStderr(logWriter)

	stateURL := fmt.Sprintf("%s/api/v1/state/%s", hubBase, deploymentID.String())

	initOpts := []tfexec.InitOption{
		tfexec.BackendConfig(fmt.Sprintf("address=%s", stateURL)),
		tfexec.BackendConfig(fmt.Sprintf("lock_address=%s/lock", stateURL)),
		tfexec.BackendConfig(fmt.Sprintf("unlock_address=%s/unlock", stateURL)),
		tfexec.BackendConfig("lock_method=POST"),
		tfexec.BackendConfig("unlock_method=POST"),
		tfexec.BackendConfig(fmt.Sprintf("username=%s", targetID)),
		tfexec.BackendConfig(fmt.Sprintf("password=%s", targetSecret)),
	}

	for key, value := range backendConfig {
		if reservedBackendKeys[key] {
			logger.Warn("ignoring reserved backend config key", zap.String("key", key))
			continue
		}
		initOpts = append(initOpts, tfexec.BackendConfig(fmt.Sprintf("%s=%s", key, value)))
	}

	return &tofuExecutor{tf: tf, workspaceDir: workspaceDir, initOpts: initOpts}, nil
}

func tofuApply(ctx context.Context, deployment api.AgentDeployment, existing *AgentDeployment) (retErr error) {
	logger.Info("tofu apply",
		zap.String("deploymentId", deployment.ID.String()),
		zap.String("configUrl", deployment.TofuConfigURL),
		zap.String("configVersion", deployment.TofuConfigVersion))

	agentDeployment := NewAgentDeployment(deployment)
	agentDeployment.State = StateInstalling
	if err := SaveDeployment(*agentDeployment); err != nil {
		return fmt.Errorf("could not save deployment state: %w", err)
	}

	defer func() {
		if retErr != nil {
			agentDeployment.State = StateFailed
			_ = SaveDeployment(*agentDeployment)
		}
	}()

	workspaceDir := WorkspaceDir(deployment.ID)

	revisionChanged := existing == nil || existing.RevisionID != deployment.RevisionID
	if revisionChanged {
		if err := os.RemoveAll(workspaceDir); err != nil {
			return fmt.Errorf("could not clean workspace before apply: %w", err)
		}
	}

	if err := pullOCIArtifact(ctx, deployment, workspaceDir); err != nil {
		return fmt.Errorf("could not pull OCI artifact: %w", err)
	}

	if err := writeVarsFile(workspaceDir, deployment.TofuVars); err != nil {
		return fmt.Errorf("could not write tfvars file: %w", err)
	}

	exec, err := prepareTofuExecutor(deployment.ID, deployment.TofuBackendConfig, "tofu") //nolint:contextcheck
	if err != nil {
		return err
	}

	logger.Info("running tofu init", zap.String("deploymentId", deployment.ID.String()))
	if err := exec.tf.Init(ctx, exec.initOpts...); err != nil {
		return fmt.Errorf("tofu init failed: %w", err)
	}

	planFile := filepath.Join(workspaceDir, "plan.out")
	logger.Info("running tofu plan", zap.String("deploymentId", deployment.ID.String()))
	hasChanges, err := exec.tf.Plan(ctx, tfexec.Out(planFile))
	if err != nil {
		return fmt.Errorf("tofu plan failed: %w", err)
	}

	if hasChanges {
		logger.Info("changes detected, running tofu apply", zap.String("deploymentId", deployment.ID.String()))
		if err := exec.tf.Apply(ctx, tfexec.DirOrPlan(planFile)); err != nil {
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

	exec, err := prepareTofuExecutor(deployment.ID, deployment.TofuBackendConfig, "tofu-destroy") //nolint:contextcheck
	if err != nil {
		return err
	}

	logger.Info("running tofu init for destroy", zap.String("deploymentId", deployment.ID.String()))
	if err := exec.tf.Init(ctx, exec.initOpts...); err != nil {
		return fmt.Errorf("tofu init for destroy failed: %w", err)
	}

	logger.Info("running tofu destroy", zap.String("deploymentId", deployment.ID.String()))
	if err := exec.tf.Destroy(ctx); err != nil {
		return fmt.Errorf("tofu destroy failed: %w", err)
	}

	if err := os.RemoveAll(exec.workspaceDir); err != nil {
		logger.Error("could not clean up workspace", zap.Error(err), zap.String("workspaceDir", exec.workspaceDir))
	}

	return nil
}

func writeVarsFile(workspaceDir string, vars map[string]any) error {
	if len(vars) == 0 {
		return nil
	}

	for key := range vars {
		if !validHCLIdentifier.MatchString(key) {
			return fmt.Errorf("invalid variable name: %q", key)
		}
	}

	data, err := json.MarshalIndent(vars, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal variables: %w", err)
	}

	varsPath := filepath.Join(workspaceDir, "terraform.tfvars.json")
	return os.WriteFile(varsPath, data, 0o600)
}
