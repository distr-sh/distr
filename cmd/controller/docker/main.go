package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os/signal"
	"slices"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/controllerauth"
	"github.com/distr-sh/distr/internal/controllercheck"
	"github.com/distr-sh/distr/internal/controllerclient"
	"github.com/distr-sh/distr/internal/controllerenv"
	"github.com/distr-sh/distr/internal/controllerlogging"
	"github.com/distr-sh/distr/internal/deploymenttargetlogs"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	dockercommand "github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	composeapi "github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	client         = util.Require(controllerclient.NewFromEnv(logger))
	dockerCli      = util.Require(dockercommand.NewDockerCli())
	composeService composeapi.Compose
	health         = controllercheck.NewServer(time.Hour)
	logWatcher     = NewLogsWatcher(30 * time.Second)
	logCollector   = &deploymenttargetlogs.BufferedCollector{}
)

func init() {
	logCollector.Delegate = client
	platformLoggingCore.Collector = logCollector
	controllerlogging.Redirect(logger)
	if controllerenv.ControllerVersionID == "" {
		logger.Warn("DISTR_CONTROLLER_VERSION_ID is not set. self updates will be disabled")
	}
	util.Must(dockerCli.Initialize(flags.NewClientOptions()))
	composeService = util.Require(compose.NewComposeService(dockerCli))
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

	logger.Info("docker controller is starting",
		zap.String("version", buildconfig.Version()),
		zap.String("commit", buildconfig.Commit()),
		zap.Bool("release", buildconfig.IsRelease()))

	go func() {
		if err := startHealthServer(); err != nil {
			logger.Warn("health server error", zap.Error(err))
		}
	}()

	mainLoop(ctx)

	logger.Info("shutting down")
}

func mainLoop(ctx context.Context) {
	tick := time.Tick(controllerenv.Interval)
	logsGoroutine := util.NewToggleableGoroutine(logWatcher.Watch)
	deploymentMetricsGoroutine := util.NewToggleableGoroutine(watchDeploymentMetrics)
	imageDiskUsageGoroutine := util.NewToggleableGoroutine(watchImageDiskUsage)

	go watchStatus(ctx)

loop:
	for ctx.Err() == nil {
		select {
		case <-tick:
		case <-ctx.Done():
			break loop
		}

		health.Heartbeat()

		if resource, err := client.Resource(ctx); err != nil {
			logger.Error("failed to get resource", zap.Error(err))
		} else {
			publishTargetRevisionIDs(resource.Deployments)

			if selfUpdateIfRequired(ctx, *resource) {
				continue
			}

			logWatcher.SetLogsAfter(resource.DeploymentLogsAfter)
			logsGoroutine.GoOrCancel(ctx, resource.DeploymentLogsEnabled)

			if resource.MetricsEnabled {
				startMetrics(ctx)
			} else {
				stopMetrics(ctx)
			}
			deploymentMetricsGoroutine.GoOrCancel(ctx, resource.MetricsEnabled)
			imageDiskUsageGoroutine.GoOrCancel(ctx, resource.MetricsEnabled)

			deployments, err := GetExistingDeployments()
			if err != nil {
				logger.Error("could not get existing deployments", zap.Error(err))
			} else {
				cleanupOldDeployments(ctx, *resource, slices.Collect(maps.Values(deployments)))
			}

			if len(resource.Deployments) == 0 {
				logger.Info("no deployment in resource response")
				continue
			}

			for _, deployment := range resource.Deployments {
				applyDeployment(ctx, deployment, deployments)
			}
		}
	}
}

func applyDeployment(
	ctx context.Context,
	deployment api.ControllerDeployment,
	existing map[uuid.UUID]ControllerDeployment,
) {
	if deployment.DockerType == nil {
		logger.Error("cannot apply deployment because docker type is nil",
			zap.Any("deploymentRevisionId", deployment.RevisionID))
		return
	}

	var controllerDeployment *ControllerDeployment
	if d, ok := existing[deployment.ID]; ok {
		controllerDeployment = &d
	}

	if !requiresApply(deployment.RevisionID, controllerDeployment) {
		if *deployment.DockerType == types.DockerTypeCompose {
			if err := EnsureComposeProjectDir(deployment); err != nil {
				logger.Warn("could not write compose project directory", zap.Error(err))
			}
		}
		return
	}

	if _, err := controllerauth.EnsureAuth(ctx, client.RawToken(), deployment); err != nil {
		logger.Error("docker auth error", zap.Error(err))
		sendApplyStatus(ctx, deployment, "", err)
		return
	}

	var previousDeploymentImages []string
	if controllerDeployment != nil {
		if images, err := GetDeploymentImages(ctx, *controllerDeployment); err != nil {
			logger.Error("failed to get old images", zap.Error(err))
		} else {
			previousDeploymentImages = images
		}
	}

	updateStatus, stopProgress := sendProgressInterval(ctx, deployment)
	defer stopProgress()
	appliedDeployment, status, err := DockerEngineApply(ctx, deployment, controllerDeployment, updateStatus)
	var restartErr error
	if err == nil && deployment.ForceRestart {
		restartErr = RunDockerRestart(ctx, *appliedDeployment)
	}

	// The final report has to arrive before the saved state lets the status watcher report the new revision's
	// health, which it would overwrite otherwise. A failed restart is not retried, so it does not fail the apply.
	stopProgress()
	sendApplyStatus(ctx, deployment, status, errors.Join(err, restartErr))
	if appliedDeployment == nil {
		return
	}
	SaveAppliedDeployment(appliedDeployment, err)

	if err == nil && deployment.ImageCleanupEnabled {
		if delErr := DeleteImages(ctx, previousDeploymentImages); delErr != nil {
			logger.Warn("failed to delete old images", zap.Error(delErr))
		}
	}
}

func requiresApply(targetRevisionID uuid.UUID, current *ControllerDeployment) bool {
	return current == nil ||
		current.RevisionID != targetRevisionID ||
		current.State == StateFailed ||
		current.State == StateProgressing
}

func publishTargetRevisionIDs(deployments []api.ControllerDeployment) {
	targets := make(map[uuid.UUID]uuid.UUID, len(deployments))
	for _, deployment := range deployments {
		targets[deployment.ID] = deployment.RevisionID
	}
	targetRevisionIDs.Store(&targets)
}

func sendApplyStatus(ctx context.Context, deployment api.ControllerDeployment, status string, err error) {
	if err != nil {
		err = client.StatusWithError(ctx, deployment, err)
	} else {
		err = client.Status(ctx, deployment, types.DeploymentStatusTypeProgressing, status)
	}

	if err != nil {
		logger.Error("failed to send status", zap.Error(err))
	}
}

// sendProgressInterval reports progressing until the returned stop function is called, which returns only once
// no further report can be sent.
func sendProgressInterval(ctx context.Context, deployment api.ControllerDeployment) (func(string), func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	var status atomic.Value
	status.Store("initializing")

	sendProgress := func() {
		err := client.Status(ctx,
			deployment, types.DeploymentStatusTypeProgressing, status.Load().(string))
		if err != nil {
			logger.Warn("error updating status", zap.Error(err))
		}
	}

	sendProgress()

	go func() {
		defer close(done)
		tick := time.Tick(controllerenv.ProgressingInterval)
		for {
			select {
			case <-ctx.Done():
				logger.Debug("stop sending progress updates")
				return
			case <-tick:
				logger.Info("sending progress update")
				sendProgress()
			}
		}
	}()

	stop := func() {
		cancel()
		<-done
	}
	return func(s string) { status.Store(s) }, stop
}

func startHealthServer() error {
	err := http.ListenAndServe("127.0.0.1:8765", health)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func selfUpdateIfRequired(ctx context.Context, resource api.ControllerResource) bool {
	if controllerenv.ControllerVersionID != "" {
		if controllerenv.ControllerVersionID != resource.Version.ID.String() {
			logger.Info("controller version has changed. starting self-update")
			if err := RunControllerSelfUpdate(ctx); err != nil {
				logger.Error("self update failed", zap.Error(err))
				// TODO: Support status without revision ID?
				if len(resource.Deployments) > 0 {
					if err := client.StatusWithError(ctx, resource.Deployments[0], err); err != nil {
						logger.Error("failed to send status", zap.Error(err))
					}
				}
			} else {
				logger.Info("self-update has been applied")
				return true
			}
		} else {
			logger.Debug("controller version is up to date")
		}
	}
	return false
}

func cleanupOldDeployments(ctx context.Context, resource api.ControllerResource, deployments []ControllerDeployment) {
	for _, deployment := range deployments {
		resourceHasExistingDeployment := slices.ContainsFunc(
			resource.Deployments,
			func(d api.ControllerDeployment) bool { return d.ID == deployment.ID },
		)
		if !resourceHasExistingDeployment {
			logger.Info("uninstalling old deployment", zap.String("id", deployment.ID.String()))

			deploymentImages, err := GetDeploymentImages(ctx, deployment)
			if err != nil {
				logger.Error("could not get images for old deployment", zap.Error(err))
			}

			if err := DockerEngineUninstall(ctx, deployment); err != nil {
				logger.Warn("could not uninstall deployment", zap.Error(err))
			} else if err := DeleteImages(ctx, deploymentImages); err != nil {
				logger.Warn("could not delete images for old deployment", zap.Error(err))
			}

			if err := DeleteDeployment(deployment); err != nil {
				logger.Warn("could not delete deployment", zap.Error(err))
			}

			if err := DeleteComposeProjectDir(deployment.ID); err != nil {
				logger.Warn("could not delete compose project directory", zap.Error(err))
			}

			logWatcher.CleanupLogsTimestamps(deployment)
		}
	}
}
