package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/controllerauth"
	"github.com/distr-sh/distr/internal/controllercheck"
	"github.com/distr-sh/distr/internal/controllerclient"
	"github.com/distr-sh/distr/internal/controllerenv"
	"github.com/distr-sh/distr/internal/deploymenttargetlogs"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"helm.sh/helm/v4/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	applyconfigurationscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	platformLoggingCore = &deploymenttargetlogs.Core{Encoder: zapcore.NewConsoleEncoder(func() zapcore.EncoderConfig {
		cfg := zap.NewDevelopmentEncoderConfig()
		cfg.TimeKey = ""
		cfg.LevelKey = ""
		return cfg
	}())}
	logger = util.Require(zap.NewDevelopment(
		zap.WrapCore(func(c zapcore.Core) zapcore.Core {
			// Platform logging should use the same logging level as the base core
			platformLoggingCore.LevelEnabler = c
			return zapcore.NewTee(c, platformLoggingCore)
		}),
	))
	controllerClient     = util.Require(controllerclient.NewFromEnv(logger))
	health               = controllercheck.NewServer(time.Hour)
	k8sConfigFlags       = genericclioptions.NewConfigFlags(true)
	k8sClient            = util.Require(kubernetes.NewForConfig(util.Require(k8sConfigFlags.ToRESTConfig())))
	metricsClientSet     = util.Require(metricsv.NewForConfig(util.Require(k8sConfigFlags.ToRESTConfig())))
	k8sDynamicClient     = util.Require(dynamic.NewForConfig(util.Require(k8sConfigFlags.ToRESTConfig())))
	k8sRestMapper        = util.Require(k8sConfigFlags.ToRESTMapper())
	controllerConfigDirs []string
	logCollector         = &deploymenttargetlogs.BufferedCollector{}
	// controllerNamespace is published by the main loop for the goroutines that have no access to the resource.
	controllerNamespace atomic.Pointer[string]
)

func init() {
	logCollector.Delegate = controllerClient
	platformLoggingCore.Collector = logCollector
	if controllerenv.ControllerVersionID == "" {
		logger.Warn("DISTR_CONTROLLER_VERSION_ID is not set. self updates will be disabled")
	}
	if s := controllerenv.Get("CONFIG_DIRS"); s != "" {
		controllerConfigDirs = slices.DeleteFunc(
			strings.Split(s, "\n"),
			func(s string) bool { return strings.TrimSpace(s) == "" },
		)
	}
}

func main() {
	defer func() {
		if err := logger.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
			fmt.Println(err)
		}
	}()

	defer func() {
		if err := logCollector.Stop(); err != nil {
			fmt.Println(err)
		}
	}()

	defer func() {
		if reason := recover(); reason != nil {
			logger.Panic("controller panic", zap.Any("reason", reason))
		}
	}()

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)

	context.AfterFunc(ctx, func() { logger.Info("shutdown signal received") })

	logger.Info("kubernetes controller is starting",
		zap.String("version", buildconfig.Version()),
		zap.String("commit", buildconfig.Commit()),
		zap.Bool("release", buildconfig.IsRelease()))

	go func() {
		if err := startHealthServer(); err != nil {
			logger.Warn("health server error", zap.Error(err))
		}
	}()

	go func() {
		logger.Info("start config watch")
		if err := watchConfigDirs(controllerConfigDirs); err != nil {
			logger.Error("config watch failed", zap.Error(err))
		} else {
			logger.Warn("config watch stopped")
		}
	}()

	var logsWatcher *logsWatcher
	var logsGoroutine *util.ToggleableGoroutine
	metricsGoroutine := util.NewToggleableGoroutine(watchMetrics)
	deploymentMetricsGoroutine := util.NewToggleableGoroutine(watchDeploymentMetrics)

	go watchStatus(ctx)

	tick := time.Tick(controllerenv.Interval)

	for ctx.Err() == nil {
		select {
		case <-tick:
		case <-ctx.Done():
			continue
		}

		health.Heartbeat()

		if changed, err := controllerClient.ReloadFromEnv(); err != nil {
			logger.Error("controller client config reload failed", zap.Error(err))
		} else if changed {
			logger.Info("controller client config reloaded")
		} else {
			logger.Debug("controller client config unchanged")
		}

		res, err := controllerClient.Resource(ctx)
		if err != nil {
			logger.Error("could not get resource", zap.Error(err))
			continue
		}

		if runSelfUpdateIfNeeded(ctx, res.Namespace, res.Version) {
			continue
		}

		if logsWatcher == nil {
			logsWatcher = NewLogsWatcher(res.Namespace, 30*time.Second)
			logsGoroutine = util.NewToggleableGoroutine(logsWatcher.Watch)
		}
		logsWatcher.SetNamespace(res.Namespace)
		logsWatcher.SetLogsAfter(res.DeploymentLogsAfter)
		logsGoroutine.GoOrCancel(ctx, res.DeploymentLogsEnabled)
		controllerNamespace.Store(&res.Namespace)
		publishTargetRevisionIDs(res.Deployments)
		metricsGoroutine.GoOrCancel(ctx, res.MetricsEnabled)
		deploymentMetricsGoroutine.GoOrCancel(ctx, res.MetricsEnabled)

		existingDeployments, err := GetExistingDeployments(ctx, res.Namespace)
		if err != nil {
			logger.Error("could not get existing deployments", zap.Error(err))
			continue
		}

		for _, existing := range existingDeployments {
			// Check if the deployment ID matches, but fall back to checking the release name if the controller
			// deployment is missing the ID. This has the disadvantage that we would miss if a deployment is
			// deleted and recreated with the same name very quickly.
			resourceHasExistingDeployment := slices.ContainsFunc(
				res.Deployments,
				func(depl api.ControllerDeployment) bool { return isSameDeployment(existing, depl) },
			)
			if !resourceHasExistingDeployment {
				logger.Info("uninstalling orphan deployment", zap.String("id", existing.ID.String()))
				if err := RunHelmUninstall(ctx, res.Namespace, existing.ReleaseName); err != nil {
					logger.Warn("could not uninstall old deployment", zap.Error(err))
				} else if err := DeleteDeployment(ctx, res.Namespace, existing); err != nil {
					logger.Warn("could not delete old ControllerDeployment resource", zap.Error(err))
				}
				registryAuthErrors.Delete(existing.ID)
			}
		}

		if len(res.Deployments) == 0 {
			logger.Info("no deployment in resource response")
			continue
		}

		for _, deployment := range res.Deployments {
			var currentDeployment *ControllerDeployment
			for _, existing := range existingDeployments {
				if isSameDeployment(existing, deployment) {
					currentDeployment = &existing
					break
				}
			}
			if err := verifyLatestHelmRelease(ctx, res.Namespace, deployment, currentDeployment); err != nil {
				if errors.Is(err, driver.ErrReleaseNotFound) {
					logger.Info("current helm release does not exist")
				} else if !requiresInstallOrUpgrade(deployment.RevisionID, currentDeployment) {
					logger.Warn("helm release differs from the one deployed by the agent", zap.Error(err))
				} else {
					logger.Warn("refusing to install or update", zap.Error(err))
					pushErrorStatus(ctx, deployment, err)
					continue
				}
			}

			runInstallOrUpgrade(ctx, res.Namespace, deployment, currentDeployment)
		}
	}

	logger.Info("shutting down")
}

func runSelfUpdateIfNeeded(ctx context.Context, namespace string, targetVersion types.ControllerVersion) bool {
	if controllerenv.ControllerVersionID == "" {
		logger.Debug("controller version is not set")
		return false
	}

	if controllerenv.ControllerVersionID == targetVersion.ID.String() {
		logger.Debug("controller version is up to date")
		return false
	}

	logger.Info("controller version has changed. starting self-update")

	manifest, err := controllerClient.Manifest(ctx)
	if err != nil {
		logger.Error("error fetching controller manifest", zap.Error(err))
		return false
	}

	parsedManifest, err := DecodeResourceYaml(manifest)
	if err != nil {
		logger.Error("error parsing controller manifest", zap.Error(err))
		return false
	}

	// add a pretty generous timeout just to prevent waiting forever in worst case
	applyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
	defer cancel()
	if err := ApplyResources(applyCtx, namespace, parsedManifest); err != nil {
		logger.Error("error applying controller manifest", zap.Error(err))
		return false
	}

	logger.Info("self-update has been applied")
	return true
}

func verifyLatestHelmRelease(
	ctx context.Context,
	namespace string,
	deployment api.ControllerDeployment,
	currentDeployment *ControllerDeployment,
) error {
	if latestRelease, err := GetLatestHelmRelease(ctx, namespace, deployment); err != nil {
		return fmt.Errorf("could not get latest helm revision: %w", err)
	} else if currentDeployment == nil {
		return fmt.Errorf("helm release %v already exists but was not created by the controller", latestRelease.Name())
	} else if currentDeployment.HelmRevision != nil && *currentDeployment.HelmRevision != latestRelease.Version() {
		msg := fmt.Sprintf("actual helm revision for %v (%v) is different from latest deployed by controller",
			latestRelease.Name(), latestRelease.Version())
		if currentDeployment.HelmRevision != nil {
			msg += fmt.Sprintf(" (%v)", *currentDeployment.HelmRevision)
		} else {
			msg += " (<nil>)"
		}
		if deployment.IgnoreRevisionSkew {
			logger.Warn(msg)
			return nil
		} else {
			return errors.New(msg)
		}
	} else {
		return nil
	}
}

func runInstallOrUpgrade(
	ctx context.Context,
	namespace string,
	deployment api.ControllerDeployment,
	currentDeployment *ControllerDeployment,
) {
	progress := Progress(deployment)

	authErr := ensureRegistryAuth(ctx, namespace, deployment)
	registryAuthErrors.Set(deployment, authErr)
	if authErr != nil {
		logger.Error("registry auth error", zap.Error(authErr))
		// The status watcher reports the error for the applied revision, since it would otherwise overwrite it.
		if currentDeployment == nil || currentDeployment.AppliedRevisionID() != deployment.RevisionID {
			pushErrorStatus(ctx, deployment, authErr)
		}
	}

	if !requiresInstallOrUpgrade(deployment.RevisionID, currentDeployment) {
		logger.Debug("no action required")
	} else if currentDeployment == nil || currentDeployment.HelmRevision == nil {
		var installed *ControllerDeployment
		err := progress.Run(ctx, func() error {
			var err error
			if installed, err = RunHelmInstall(ctx, namespace, deployment, currentDeployment); err != nil {
				return fmt.Errorf("helm install failed: %w", err)
			}
			return nil
		})
		if err != nil {
			logger.Error("install error", zap.Error(err))
			pushErrorStatus(ctx, deployment, fmt.Errorf("install error: %w", err))
		} else {
			logger.Info("helm install succeeded")
			pushRunningStatus(ctx, deployment, "helm install succeeded")
			saveReadyDeployment(ctx, namespace, installed)
		}
	} else {
		successMessage := "helm upgrade succeeded"
		var upgraded *ControllerDeployment
		var restartErr error
		err := progress.Run(ctx, func() error {
			var err error
			if upgraded, err = RunHelmUpgrade(ctx, namespace, deployment, *currentDeployment); err != nil {
				return fmt.Errorf("helm upgrade failed: %w", err)
			}
			if deployment.ForceRestart {
				if restartErr = ForceRestart(ctx, namespace, *upgraded); restartErr == nil {
					successMessage += "; force restart succeeded"
				}
			}
			return nil
		})
		if err != nil {
			logger.Error("upgrade error", zap.Error(err))
			pushErrorStatus(ctx, deployment, fmt.Errorf("upgrade error: %w", err))
			return
		}
		// The final report has to arrive before the saved state lets the status watcher report the new revision's
		// health, which it would overwrite otherwise. A failed restart is not retried, so it does not fail the upgrade.
		if restartErr != nil {
			logger.Error("force restart error", zap.Error(restartErr))
			pushErrorStatus(ctx, deployment, fmt.Errorf("%v; force restart error: %w", successMessage, restartErr))
		} else {
			logger.Info(successMessage)
			pushRunningStatus(ctx, deployment, successMessage)
		}
		saveReadyDeployment(ctx, namespace, upgraded)
	}
}

func requiresInstallOrUpgrade(targetRevisionID uuid.UUID, currentDeployment *ControllerDeployment) bool {
	return currentDeployment == nil ||
		currentDeployment.HelmRevision == nil ||
		currentDeployment.RevisionID != targetRevisionID ||
		currentDeployment.State == StateProgressing
}

func publishTargetRevisionIDs(deployments []api.ControllerDeployment) {
	targets := make(map[uuid.UUID]uuid.UUID, len(deployments))
	for _, deployment := range deployments {
		targets[deployment.ID] = deployment.RevisionID
	}
	targetRevisionIDs.Store(&targets)
}

type progressStatusRunner struct {
	deployment api.ControllerDeployment
}

func Progress(deployment api.ControllerDeployment) *progressStatusRunner {
	return &progressStatusRunner{deployment: deployment}
}

// Run reports progressing while f runs and returns only once no further progressing report can be sent.
func (psr *progressStatusRunner) Run(ctx context.Context, f func() error) error {
	progressCtx, progressCancel := context.WithCancel(ctx)
	done := make(chan struct{})
	defer func() {
		progressCancel()
		<-done
	}()

	pushProgressingStatus(ctx, psr.deployment)

	go func(ctx context.Context) {
		defer close(done)
		tick := time.Tick(controllerenv.ProgressingInterval)
		for {
			select {
			case <-ctx.Done():
				logger.Debug("stop sending progress updates")
				return
			case <-tick:
				logger.Info("sending progress update")
				pushProgressingStatus(ctx, psr.deployment)
			}
		}
	}(progressCtx)

	return f()
}

func pushRunningStatus(ctx context.Context, deployment api.ControllerDeployment, status string) {
	if err := controllerClient.Status(ctx, deployment, types.DeploymentStatusTypeRunning, status); err != nil {
		logger.Warn("status push failed", zap.Error(err))
	}
}

func pushProgressingStatus(ctx context.Context, deployment api.ControllerDeployment) {
	if err := controllerClient.Status(
		ctx,
		deployment,
		types.DeploymentStatusTypeProgressing,
		"helm operation in progress",
	); err != nil {
		logger.Warn("status push failed", zap.Error(err))
	}
}

func pushErrorStatus(ctx context.Context, deployment api.ControllerDeployment, err error) {
	if err := controllerClient.Status(ctx, deployment, types.DeploymentStatusTypeError, err.Error()); err != nil {
		logger.Warn("status push failed", zap.Error(err))
	}
}

func ensureRegistryAuth(ctx context.Context, namespace string, deployment api.ControllerDeployment) error {
	if _, err := controllerauth.EnsureAuth(ctx, controllerClient.RawToken(), deployment); err != nil {
		return fmt.Errorf("failed to ensure docker auth: %w", err)
	} else if err := ensureImagePullSecret(ctx, namespace, deployment); err != nil {
		return fmt.Errorf("failed to ensure image pull secret: %w", err)
	}
	return nil
}

func ensureImagePullSecret(ctx context.Context, namespace string, deployment api.ControllerDeployment) error {
	// It's easiest to simply copy the docker config from the file previously created by [controllerauth.EnsureAuth].
	// However, be aware that this will not work when running the controller locally when a docker credential helper
	// is installed.
	dockerConfigPath := controllerauth.DockerConfigPath(deployment)
	dockerConfigData, err := os.ReadFile(dockerConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read docker config from %v: %w", dockerConfigPath, err)
	}
	secretName := PullSecretName(deployment.ReleaseName)
	secretCfg := applyconfigurationscorev1.Secret(secretName, namespace)
	secretCfg.WithType("kubernetes.io/dockerconfigjson")
	secretCfg.WithData(map[string][]byte{
		".dockerconfigjson": dockerConfigData,
	})
	_, err = k8sClient.CoreV1().Secrets(namespace).Apply(
		ctx,
		secretCfg,
		metav1.ApplyOptions{Force: true, FieldManager: fieldManager},
	)
	if err != nil {
		return fmt.Errorf("failed to apply secret resource %v: %w", secretName, err)
	}
	return nil
}

func startHealthServer() error {
	err := http.ListenAndServe(":8765", health)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func watchConfigDirs(dirs []string) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	for _, dir := range dirs {
		if err := w.Add(dir); err != nil {
			return err
		}
	}
	for {
		select {
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			return err
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if event.Op != fsnotify.Rename && event.Op != fsnotify.Write {
				continue
			}
			for _, dir := range dirs {
				logger := logger.With(zap.String("dir", dir))
				entries, err := os.ReadDir(dir)
				if err != nil {
					logger.Warn("read dir failed", zap.Error(err))
					continue
				}
				for _, e := range entries {
					logger := logger.With(zap.String("entry", e.Name()))
					if e.IsDir() {
						continue
					}
					if data, err := os.ReadFile(path.Join(dir, e.Name())); err != nil {
						logger.Warn("could not update config param", zap.Error(err))
					} else {
						logger.Debug("setting env variable from file", zap.String("value", string(data)))
						os.Setenv(e.Name(), string(data))
					}
				}
			}
		}
	}
}

func isSameDeployment(existingDeployment ControllerDeployment, resourceDeployment api.ControllerDeployment) bool {
	return (existingDeployment.ID != uuid.Nil && existingDeployment.ID == resourceDeployment.ID) ||
		(existingDeployment.ID == uuid.Nil && resourceDeployment.ReleaseName == existingDeployment.ReleaseName)
}
