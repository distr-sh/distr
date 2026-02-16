package main

import (
	"context"
	"fmt"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentauth"
	"github.com/distr-sh/distr/internal/agentenv"
	"github.com/distr-sh/distr/internal/util"
	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart"
	"helm.sh/helm/v4/pkg/chart/loader"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/kube"
	"helm.sh/helm/v4/pkg/registry"
	"helm.sh/helm/v4/pkg/release"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var helmEnvSettings = cli.New()

func GetHelmActionConfig(
	ctx context.Context,
	namespace string,
	deployment *api.AgentDeployment,
) (*action.Configuration, error) {
	var cfg action.Configuration
	cfg.SetLogger(slogzap.Option{Logger: logger.With(zap.String("component", "helm"))}.NewZapHandler())

	if deployment != nil {
		var clientOpts []registry.ClientOption

		if agentenv.DistrRegistryPlainHTTP {
			clientOpts = append(clientOpts, registry.ClientOptPlainHTTP())
		}

		if authorizer, err := agentauth.EnsureAuth(ctx, agentClient.RawToken(), *deployment); err != nil {
			return nil, err
		} else {
			clientOpts = append(clientOpts, registry.ClientOptAuthorizer(*authorizer))
		}

		if rc, err := registry.NewClient(clientOpts...); err != nil {
			return nil, err
		} else {
			cfg.RegistryClient = rc
		}
	}

	if err := cfg.Init(k8sConfigFlags, namespace, "secret"); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func GetLatestHelmRelease(
	ctx context.Context,
	namespace string,
	deployment api.AgentDeployment,
) (release.Accessor, error) {
	cfg, err := GetHelmActionConfig(ctx, namespace, nil)
	if err != nil {
		return nil, err
	}

	// Get returns the latest revision by default
	if releaser, err := action.NewGet(cfg).Run(deployment.ReleaseName); err != nil {
		return nil, err
	} else {
		return release.NewAccessor(releaser)
	}
}

func RunHelmPreflight(
	action *action.ChartPathOptions,
	deployment api.AgentDeployment,
) (chart.Charter, error) {
	chartName := deployment.ChartName
	action.Version = deployment.ChartVersion
	if registry.IsOCI(deployment.ChartUrl) {
		chartName = deployment.ChartUrl
	} else {
		action.RepoURL = deployment.ChartUrl
	}
	if chartPath, err := action.LocateChart(chartName, helmEnvSettings); err != nil {
		return nil, fmt.Errorf("could not locate chart: %w", err)
	} else if charter, err := loader.Load(chartPath); err != nil {
		return nil, fmt.Errorf("chart loading failed: %w", err)
	} else {
		addImagePullSecretToValues(deployment.ReleaseName, deployment.Values)
		return charter, nil
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
	installAction.WaitStrategy = kube.StatusWatcherStrategy
	installAction.DryRunStrategy = action.DryRunNone
	installAction.RollbackOnFailure = true
	installAction.Namespace = namespace
	installAction.PlainHTTP = agentenv.DistrRegistryPlainHTTP

	c, err := RunHelmPreflight(&installAction.ChartPathOptions, deployment)
	if err != nil {
		return nil, fmt.Errorf("helm preflight failed: %w", err)
	}

	agentDeployment := NewAgentDeployment(deployment)
	agentDeployment.State = StateProgressing
	if err := SaveDeployment(ctx, namespace, agentDeployment); err != nil {
		logger.Warn("failed to save deployment before install", zap.Error(err))
	}

	releaser, err := installAction.RunWithContext(ctx, c, deployment.Values)
	if err != nil {
		err = fmt.Errorf("helm install failed: %w", err)
		agentDeployment.State = StateFailed
	} else if acc, err := release.NewAccessor(releaser); err != nil {
		return nil, fmt.Errorf("failed to create release accessor: %w", err)
	} else {
		agentDeployment.State = StateReady
		agentDeployment.HelmRevision = util.PtrTo(acc.Version())
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
	upgradeAction.Timeout = 5 * time.Minute
	upgradeAction.WaitStrategy = kube.StatusWatcherStrategy
	upgradeAction.DryRunStrategy = action.DryRunNone
	upgradeAction.CleanupOnFail = true
	upgradeAction.RollbackOnFailure = true
	upgradeAction.Namespace = namespace
	upgradeAction.PlainHTTP = agentenv.DistrRegistryPlainHTTP

	chart, err := RunHelmPreflight(&upgradeAction.ChartPathOptions, deployment)
	if err != nil {
		return nil, fmt.Errorf("helm preflight failed: %w", err)
	}

	releaser, err := upgradeAction.RunWithContext(ctx, deployment.ReleaseName, chart, deployment.Values)
	if err != nil {
		return nil, fmt.Errorf("helm upgrade failed: %w", err)
	}

	acc, err := release.NewAccessor(releaser)
	if err != nil {
		return nil, fmt.Errorf("failed to create release accessor: %w", err)
	}

	agentDeployment := NewAgentDeployment(deployment)
	agentDeployment.State = StateReady
	agentDeployment.HelmRevision = util.PtrTo(acc.Version())
	if err := SaveDeployment(ctx, namespace, agentDeployment); err != nil {
		logger.Warn("failed to save deployment after upgrade", zap.Error(err))
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
	uninstallAction.WaitStrategy = kube.StatusWatcherStrategy
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
	if releaser, err := getAction.Run(releaseName); err != nil {
		return nil, err
	} else if acc, err := release.NewAccessor(releaser); err != nil {
		return nil, err
	} else {
		// decode the release manifests which is represented as multi-document YAML
		return DecodeResourceYaml([]byte(acc.Manifest()))
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
