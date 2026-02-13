package main

import (
	"context"
	"fmt"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentauth"
	"github.com/distr-sh/distr/internal/agentenv"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	helmEnvSettings       = cli.New()
	helmActionConfigCache = make(map[string]*action.Configuration)
)

func GetHelmActionConfig(
	ctx context.Context,
	namespace string,
	deployment *api.AgentDeployment,
) (*action.Configuration, error) {
	if cfg, ok := helmActionConfigCache[namespace]; ok {
		return cfg, nil
	}

	var cfg action.Configuration
	var clientOpts []registry.ClientOption
	if agentenv.DistrRegistryPlainHTTP {
		clientOpts = append(clientOpts, registry.ClientOptPlainHTTP())
	}
	if deployment != nil {
		if authorizer, err := agentauth.EnsureAuth(ctx, agentClient.RawToken(), *deployment); err != nil {
			return nil, err
		} else if rc, err := registry.NewClient(
			append(clientOpts, registry.ClientOptAuthorizer(*authorizer))...,
		); err != nil {
			return nil, err
		} else {
			cfg.RegistryClient = rc
		}
	}
	if err := cfg.Init(
		k8sConfigFlags,
		namespace,
		"secret",
		func(format string, v ...any) { logger.Sugar().Debugf(format, v...) },
	); err != nil {
		return nil, err
	} else {
		return &cfg, nil
	}
}

func GetLatestHelmRelease(
	ctx context.Context,
	namespace string,
	deployment api.AgentDeployment,
) (*release.Release, error) {
	cfg, err := GetHelmActionConfig(ctx, namespace, nil)
	if err != nil {
		return nil, err
	}
	// Get returns the latest revision by default
	return action.NewGet(cfg).Run(deployment.ReleaseName)
}

func RunHelmPreflight(
	action *action.ChartPathOptions,
	deployment api.AgentDeployment,
) (*chart.Chart, error) {
	chartName := deployment.ChartName
	action.Version = deployment.ChartVersion
	if registry.IsOCI(deployment.ChartUrl) {
		chartName = deployment.ChartUrl
	} else {
		action.RepoURL = deployment.ChartUrl
	}
	if chartPath, err := action.LocateChart(chartName, helmEnvSettings); err != nil {
		return nil, fmt.Errorf("could not locate chart: %w", err)
	} else if chart, err := loader.Load(chartPath); err != nil {
		return nil, fmt.Errorf("chart loading failed: %w", err)
	} else {
		addImagePullSecretToValues(deployment.ReleaseName, deployment.Values)
		return chart, nil
	}
}

func RunHelmInstall(
	ctx context.Context,
	namespace string,
	deployment api.AgentDeployment,
) (*AgentDeployment, error) {
	config, err := GetHelmActionConfig(ctx, namespace, &deployment)
	if err != nil {
		return nil, err
	}

	installAction := action.NewInstall(config)
	installAction.ReleaseName = deployment.ReleaseName

	installAction.Timeout = 5 * time.Minute
	installAction.Wait = true
	installAction.Atomic = true
	installAction.Namespace = namespace
	installAction.PlainHTTP = agentenv.DistrRegistryPlainHTTP

	chart, err := RunHelmPreflight(&installAction.ChartPathOptions, deployment)
	if err != nil {
		return nil, fmt.Errorf("helm preflight failed: %w", err)
	}

	agentDeployment := NewAgentDeployment(deployment)
	agentDeployment.State = StateProgressing
	if err := SaveDeployment(ctx, namespace, agentDeployment); err != nil {
		logger.Warn("failed to save deployment before install", zap.Error(err))
	}

	release, err := installAction.RunWithContext(ctx, chart, deployment.Values)
	if err != nil {
		err = fmt.Errorf("helm install failed: %w", err)
		agentDeployment.State = StateFailed
	} else {
		agentDeployment.State = StateReady
		agentDeployment.HelmRevision = &release.Version
	}

	if err := SaveDeployment(ctx, namespace, agentDeployment); err != nil {
		logger.Warn("failed to save deployment after install", zap.Error(err))
	}

	return &agentDeployment, err
}

func RunHelmUpgrade(
	ctx context.Context,
	namespace string,
	deployment api.AgentDeployment,
) (*AgentDeployment, error) {
	cfg, err := GetHelmActionConfig(ctx, namespace, &deployment)
	if err != nil {
		return nil, err
	}

	upgradeAction := action.NewUpgrade(cfg)
	upgradeAction.CleanupOnFail = true

	upgradeAction.Timeout = 5 * time.Minute
	upgradeAction.Wait = true
	upgradeAction.Atomic = true
	upgradeAction.Namespace = namespace
	upgradeAction.PlainHTTP = agentenv.DistrRegistryPlainHTTP

	chart, err := RunHelmPreflight(&upgradeAction.ChartPathOptions, deployment)
	if err != nil {
		return nil, fmt.Errorf("helm preflight failed: %w", err)
	}

	release, err := upgradeAction.RunWithContext(ctx, deployment.ReleaseName, chart, deployment.Values)
	if err != nil {
		return nil, fmt.Errorf("helm upgrade failed: %w", err)
	}

	agentDeployment := NewAgentDeployment(deployment)
	agentDeployment.State = StateReady
	agentDeployment.HelmRevision = &release.Version
	if err := SaveDeployment(ctx, namespace, agentDeployment); err != nil {
		logger.Warn("failed to save deployment after install", zap.Error(err))
	}

	return &agentDeployment, err
}

func RunHelmUninstall(ctx context.Context, namespace, releaseName string) error {
	config, err := GetHelmActionConfig(ctx, namespace, nil)
	if err != nil {
		return err
	}

	uninstallAction := action.NewUninstall(config)
	uninstallAction.Timeout = 5 * time.Minute
	uninstallAction.Wait = true
	uninstallAction.IgnoreNotFound = true
	if _, err := uninstallAction.Run(releaseName); err != nil {
		return fmt.Errorf("helm uninstall failed: %w", err)
	}
	return nil
}

func GetHelmManifest(ctx context.Context, namespace, releaseName string) ([]*unstructured.Unstructured, error) {
	cfg, err := GetHelmActionConfig(ctx, namespace, nil)
	if err != nil {
		return nil, err
	}
	getAction := action.NewGet(cfg)
	if release, err := getAction.Run(releaseName); err != nil {
		return nil, err
	} else {
		// decode the release manifests which is represented as multi-document YAML
		return DecodeResourceYaml([]byte(release.Manifest))
	}
}

func addImagePullSecretToValues(relaseName string, values map[string]any) {
	if s, ok := values["imagePullSecrets"].([]any); ok {
		values["imagePullSecrets"] = append(s, map[string]any{"name": PullSecretName(relaseName)})
	}
	if s, ok := values["pullSecrets"].([]any); ok {
		values["pullSecrets"] = append(s, map[string]any{"name": PullSecretName(relaseName)})
	}
	for _, v := range values {
		if m, ok := v.(map[string]any); ok {
			addImagePullSecretToValues(relaseName, m)
		}
	}
}
