package main

import (
	"context"
	"errors"
	"fmt"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/agentauth"
	"github.com/distr-sh/distr/internal/agentclient"
	"github.com/distr-sh/distr/internal/agentenv"
	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/deploymenttargetlogs"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	dockercommand "github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
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
	client         = util.Require(agentclient.NewFromEnv(logger))
	dockerCli      = util.Require(dockercommand.NewDockerCli())
	composeService = util.Require(compose.NewComposeService(dockerCli))
)

func init() {
	platformLoggingCore.Collector = &deploymenttargetlogs.BufferedCollector{Delegate: client}
	if agentenv.AgentVersionID == "" {
		logger.Warn("AgentVersionID is not set. self updates will be disabled")
	}
	util.Must(dockerCli.Initialize(flags.NewClientOptions()))
}

func main() {
	defer func() {
		if err := logger.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
			fmt.Println(err)
		}
	}()

	defer func() {
		if reason := recover(); reason != nil {
			logger.Panic("agent panic", zap.Any("reason", reason))
		}
	}()

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)

	context.AfterFunc(ctx, func() { logger.Info("shutdown signal received") })

	logger.Info("docker agent is starting",
		zap.String("version", buildconfig.Version()),
		zap.String("commit", buildconfig.Commit()),
		zap.Bool("release", buildconfig.IsRelease()))

	go NewLogsWatcher().Watch(ctx, 30*time.Second)

	mainLoop(ctx)

	logger.Info("shutting down")
}

func mainLoop(ctx context.Context) {
	tick := time.Tick(agentenv.Interval)

loop:
	for ctx.Err() == nil {
		select {
		case <-tick:
		case <-ctx.Done():
			break loop
		}

		if resource, err := client.Resource(ctx); err != nil {
			logger.Error("failed to get resource", zap.Error(err))
		} else {
			if agentenv.AgentVersionID != "" {
				if agentenv.AgentVersionID != resource.Version.ID.String() {
					logger.Info("agent version has changed. starting self-update")
					if err := RunAgentSelfUpdate(ctx); err != nil {
						logger.Error("self update failed", zap.Error(err))
						// TODO: Support status without revision ID?
						if len(resource.Deployments) > 0 {
							if err := client.StatusWithError(ctx, resource.Deployments[0].RevisionID, err); err != nil {
								logger.Error("failed to send status", zap.Error(err))
							}
						}
					} else {
						logger.Info("self-update has been applied")
						continue
					}
				} else {
					logger.Debug("agent version is up to date")
				}
			}

			if resource.MetricsEnabled {
				startMetrics(ctx)
			} else {
				stopMetrics(ctx)
			}

			deployments, err := GetExistingDeployments()
			if err != nil {
				logger.Error("could not get existing deployments", zap.Error(err))
			} else {
				for _, deployment := range deployments {
					resourceHasExistingDeployment := slices.ContainsFunc(
						resource.Deployments,
						func(d api.AgentDeployment) bool { return d.ID == deployment.ID },
					)
					if !resourceHasExistingDeployment {
						logger.Info("uninstalling old deployment", zap.String("id", deployment.ID.String()))
						if err := DockerEngineUninstall(ctx, deployment); err != nil {
							logger.Error("could not uninstall deployment", zap.Error(err))
						} else if err := DeleteDeployment(deployment); err != nil {
							logger.Error("could not delete deployment", zap.Error(err))
						}
						CleanupLogsTimestamps(deployment)
					}
				}
			}

			if len(resource.Deployments) == 0 {
				logger.Info("no deployment in resource response")
				continue
			}

			for _, deployment := range resource.Deployments {
				var agentDeployment *AgentDeployment
				var status string
				statusType := types.DeploymentStatusTypeProgressing
				_, err = agentauth.EnsureAuth(ctx, client.RawToken(), deployment)
				if err != nil {
					logger.Error("docker auth error", zap.Error(err))
				} else {
					if deployment.DockerType == nil {
						logger.Error("cannot apply deployment because docker type is nil",
							zap.Any("deploymentRevisionId", deployment.RevisionID))
						continue
					}

					if existing, ok := deployments[deployment.ID]; ok {
						agentDeployment = &existing
					}

					if agentDeployment == nil ||
						agentDeployment.RevisionID != deployment.RevisionID ||
						agentDeployment.State == StateFailed ||
						agentDeployment.State == StateProgressing {
						func() {
							progressCtx, progressCancel := context.WithCancel(ctx)
							defer progressCancel()
							go sendProgressInterval(progressCtx, deployment.RevisionID)
							agentDeployment, status, err = DockerEngineApply(ctx, deployment)
							if err == nil && deployment.ForceRestart {
								err = errors.Join(err, RunDockerRestart(ctx, *agentDeployment))
							}
						}()
					} else {
						if statusType1, statusMessage, err1 := CheckStatus(ctx, *agentDeployment); err1 != nil {
							err = errors.Join(err, err1)
						} else {
							status = statusMessage
							statusType = statusType1
						}
					}
				}

				if err != nil {
					err = client.StatusWithError(ctx, deployment.RevisionID, err)
				} else {
					err = client.Status(ctx, deployment.RevisionID, statusType, status)
				}

				if err != nil {
					logger.Error("failed to send status", zap.Error(err))
				}
			}
		}
	}
}

func sendProgressInterval(ctx context.Context, revisionID uuid.UUID) {
	tick := time.Tick(agentenv.Interval)
	for {
		select {
		case <-ctx.Done():
			logger.Debug("stop sending progress updates")
			return
		case <-tick:
			logger.Info("sending progress update")
			err := client.Status(
				ctx,
				revisionID,
				types.DeploymentStatusTypeProgressing,
				"applying docker compose…",
			)
			if err != nil {
				logger.Warn("error updating status", zap.Error(err))
			}
		}
	}
}
