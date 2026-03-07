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
	"github.com/distr-sh/distr/internal/agentclient"
	"github.com/distr-sh/distr/internal/agentenv"
	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/deploymenttargetlogs"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
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
			platformLoggingCore.LevelEnabler = c
			return zapcore.NewTee(c, platformLoggingCore)
		}),
	))
	client = util.Require(agentclient.NewFromEnv(logger))
)

func init() {
	platformLoggingCore.Collector = &deploymenttargetlogs.BufferedCollector{Delegate: client}
	if agentenv.AgentVersionID == "" {
		logger.Warn("AgentVersionID is not set. self updates will be disabled")
	}
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

	logger.Info("opentofu agent is starting",
		zap.String("version", buildconfig.Version()),
		zap.String("commit", buildconfig.Commit()),
		zap.Bool("release", buildconfig.IsRelease()))

	mainLoop(ctx)

	logger.Info("shutting down")
}

func mainLoop(ctx context.Context) {
	ticker := time.NewTicker(agentenv.Interval)
	defer ticker.Stop()

loop:
	for ctx.Err() == nil {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			break loop
		}

		resource, err := client.Resource(ctx)
		if err != nil {
			logger.Error("failed to get resource", zap.Error(err))
			continue
		}

		if agentenv.AgentVersionID != "" {
			if agentenv.AgentVersionID != resource.Version.ID.String() {
				logger.Info("agent version has changed. starting self-update")
				if err := runSelfUpdate(ctx); err != nil {
					logger.Error("self update failed", zap.Error(err))
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

		deployments, err := GetExistingDeployments()
		if err != nil {
			logger.Error("could not get existing deployments", zap.Error(err))
			continue
		}

		for _, deployment := range deployments {
			resourceHasExistingDeployment := slices.ContainsFunc(
				resource.Deployments,
				func(d api.AgentDeployment) bool { return d.ID == deployment.ID },
			)
			if !resourceHasExistingDeployment {
				logger.Info("destroying orphaned deployment", zap.String("id", deployment.ID.String()))
				if err := client.Status(
					ctx, deployment.RevisionID, types.DeploymentStatusTypeProgressing, "destroying",
				); err != nil {
					logger.Error("failed to send status", zap.Error(err))
				}
				if err := tofuDestroy(ctx, deployment); err != nil {
					logger.Error("could not destroy deployment", zap.Error(err))
					if err := client.StatusWithError(ctx, deployment.RevisionID, err); err != nil {
						logger.Error("failed to send status", zap.Error(err))
					}
				} else if err := DeleteDeployment(deployment); err != nil {
					logger.Error("could not delete deployment", zap.Error(err))
				} else {
					logger.Info("orphaned deployment destroyed", zap.String("id", deployment.ID.String()))
				}
			}
		}

		if len(resource.Deployments) == 0 {
			logger.Info("no deployment in resource response")
			continue
		}

		for _, deployment := range resource.Deployments {
			existing, hasExisting := deployments[deployment.ID]

			needsApply := !hasExisting ||
				existing.RevisionID != deployment.RevisionID ||
				existing.State != StateInstalled

			if needsApply {
				logger.Info("applying deployment",
					zap.String("id", deployment.ID.String()),
					zap.String("revisionId", deployment.RevisionID.String()))

				func() {
					progressCtx, progressCancel := context.WithCancel(ctx)
					defer progressCancel()
					go sendProgressInterval(progressCtx, deployment.RevisionID)

					if err := tofuApply(ctx, deployment); err != nil {
						logger.Error("apply failed", zap.Error(err))
						if err := client.StatusWithError(ctx, deployment.RevisionID, err); err != nil {
							logger.Error("failed to send status", zap.Error(err))
						}
					} else {
						if err := client.Status(ctx, deployment.RevisionID, types.DeploymentStatusTypeHealthy, "up to date"); err != nil {
							logger.Error("failed to send status", zap.Error(err))
						}
					}
				}()
			} else {
				if err := client.Status(ctx, deployment.RevisionID, types.DeploymentStatusTypeHealthy, "up to date"); err != nil {
					logger.Error("failed to send status", zap.Error(err))
				}
			}
		}
	}
}

func sendProgressInterval(ctx context.Context, revisionID uuid.UUID) {
	ticker := time.NewTicker(agentenv.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Debug("stop sending progress updates")
			return
		case <-ticker.C:
			logger.Info("sending progress update")
			if err := client.Status(
				ctx,
				revisionID,
				types.DeploymentStatusTypeProgressing,
				"applying opentofu configuration...",
			); err != nil {
				logger.Warn("error updating status", zap.Error(err))
			}
		}
	}
}

func runSelfUpdate(ctx context.Context) error {
	manifest, err := client.Manifest(ctx)
	if err != nil {
		return fmt.Errorf("error fetching agent manifest: %w", err)
	}
	logger.Info("received self-update manifest", zap.Int("bytes", len(manifest)))
	// Self-update mechanism for the OpenTofu agent will be implemented
	// alongside the container image build pipeline. The manifest contains
	// the compose file for restarting with the new agent version.
	return nil
}

type zapLogWriter struct {
	logger *zap.Logger
	prefix string
}

func (w *zapLogWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(w.prefix, zap.String("output", string(p)))
	return len(p), nil
}
