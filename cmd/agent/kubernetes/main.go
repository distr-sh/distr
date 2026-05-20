package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/glasskube/distr/internal/agentclient"
	"github.com/glasskube/distr/internal/util"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	interval         = 5 * time.Second
	logger           = util.Require(zap.NewDevelopment())
	agentClient      = util.Require(agentclient.NewFromEnv(logger))
	k8sConfigFlags   = genericclioptions.NewConfigFlags(true)
	k8sClient        = util.Require(kubernetes.NewForConfig(util.Require(k8sConfigFlags.ToRESTConfig())))
	k8sDynamicClient = util.Require(dynamic.NewForConfig(util.Require(k8sConfigFlags.ToRESTConfig())))
	k8sRestMapper    = util.Require(k8sConfigFlags.ToRESTMapper())
	agentVersionId   = os.Getenv("DISTR_AGENT_VERSION_ID")
)

func init() {
	if intervalStr, ok := os.LookupEnv("DISTR_INTERVAL"); ok {
		interval = util.Require(time.ParseDuration(intervalStr))
	}
	if agentVersionId == "" {
		logger.Warn("DISTR_AGENT_VERSION_ID is not set. self updates will be disabled")
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGTERM, syscall.SIGINT)
		<-sigint
		logger.Info("received termination signal")
		cancel()
	}()
	tick := time.Tick(interval)

	for ctx.Err() == nil {
		select {
		case <-tick:
		case <-ctx.Done():
			continue
		}

		res, err := agentClient.KubernetesResource(ctx)
		if err != nil {
			logger.Error("could not get resource", zap.Error(err))
			continue
		}

		if agentVersionId != "" {
			if agentVersionId != res.Version.ID.String() {
				logger.Info("agent version has changed. starting self-update")
				if manifest, err := agentClient.Manifest(ctx); err != nil {
					logger.Error("error fetching agent manifest", zap.Error(err))
				} else if parsedManifest, err := DecodeResourceYaml(manifest); err != nil {
					logger.Error("error parsing agent manifest", zap.Error(err))
				} else if err := ApplyResources(ctx, res.Namespace, parsedManifest); err != nil {
					logger.Error("error applying agent manifest", zap.Error(err))
				} else {
					logger.Info("self-update has been applied")
				}
				continue
			} else {
				logger.Debug("agent version is up to date")
			}
		}

		if res.Deployment == nil {
			// TODO: delete previous deployment if it exists?
			logger.Info("no deployment in resource response")
			continue
		}

		pushStatus := func(ctx context.Context, status string) {
			if err := agentClient.Status(ctx, res.Deployment.RevisionID, status, nil); err != nil {
				logger.Warn("status push failed", zap.Error(err))
			}
		}
		pushErrorStatus := func(ctx context.Context, error error) {
			if err := agentClient.Status(ctx, res.Deployment.RevisionID, "", error); err != nil {
				logger.Warn("status push failed", zap.Error(err))
			}
		}

		installRequired := true
		upgradeRequired := true
		var currentDeployment *AgentDeployment
		if deployments, err := GetExistingDeployments(ctx, res.Namespace); err != nil {
			logger.Error("could not get existing deployments", zap.Error(err))
			continue
		} else {
			for _, deployment := range deployments {
				if deployment.ReleaseName == res.Deployment.ReleaseName {
					currentDeployment = &deployment
					upgradeRequired = deployment.RevisionID != res.Deployment.RevisionID
				} else {
					// TODO: existing deployments should probably be handled somehow... for now we just print a warning
					logger.Sugar().Warnf("found unhandled deployment: %v", deployment.ReleaseName)
				}
			}
		}

		latestRelease, err := GetLatestHelmRelease(ctx, res.Namespace, *res.Deployment)
		if err != nil {
			if errors.Is(err, driver.ErrReleaseNotFound) {
				logger.Info("current helm release does not exist")
			} else {
				logger.Error("could not get latest helm revision", zap.Error(err))
				pushErrorStatus(ctx, err)
				continue
			}
		} else if currentDeployment != nil {
			if currentDeployment.HelmRevision != latestRelease.Version {
				msg := fmt.Sprintf("actual helm revision (%v) is different from latest deployed by agent (%v). bailing out",
					latestRelease.Version, currentDeployment.HelmRevision)
				logger.Warn(msg)
				pushErrorStatus(ctx, errors.New(msg))
				continue
			} else {
				installRequired = false
			}
		} else {
			msg := "helm release already exists but was not created by the agent. bailing out"
			logger.Warn(msg)
			pushErrorStatus(ctx, errors.New(msg))
			continue
		}

		if installRequired {
			pushStatus(ctx, "helm install in progress")
			if deployment, err := RunHelmInstall(ctx, res.Namespace, *res.Deployment); err != nil {
				logger.Error("helm upgrade failed", zap.Error(err))
				pushErrorStatus(ctx, fmt.Errorf("helm upgrade failed: %w", err))
			} else if err := SaveDeployment(ctx, res.Namespace, *deployment); err != nil {
				logger.Error("could not save latest deployment", zap.Error(err))
				pushErrorStatus(ctx, fmt.Errorf("could not save latest deployment: %w", err))
			} else {
				logger.Info("helm install succeeded")
				pushStatus(ctx, "helm install succeeded")
			}
		} else if upgradeRequired {
			pushStatus(ctx, "helm upgrade in progress")
			if deployment, err := RunHelmUpgrade(ctx, res.Namespace, *res.Deployment); err != nil {
				logger.Error("helm upgrade failed", zap.Error(err))
				// Helm Atomic rollback creates a new revision in the cluster. Sync the tracking
				// secret so the next iteration's revision check passes and the agent retries.
				if recovered, recoverErr := GetLatestHelmRelease(ctx, res.Namespace, *res.Deployment); recoverErr == nil {
					synced := AgentDeployment{
						ReleaseName:  currentDeployment.ReleaseName,
						RevisionID:   res.Deployment.RevisionID,
						HelmRevision: recovered.Version,
					}
					if saveErr := SaveDeployment(ctx, res.Namespace, synced); saveErr != nil {
						logger.Warn("could not sync tracking secret after rollback", zap.Error(saveErr))
					} else {
						logger.Info("synced tracking secret after atomic rollback", zap.Int("helmRevision", recovered.Version))
					}
				}
				pushErrorStatus(ctx, fmt.Errorf("helm upgrade failed: %w", err))
			} else if err := SaveDeployment(ctx, res.Namespace, *deployment); err != nil {
				logger.Error("could not save latest deployment", zap.Error(err))
				pushErrorStatus(ctx, fmt.Errorf("could not save latest deployment: %w", err))
			} else {
				logger.Info("helm upgrade succeeded")
				pushStatus(ctx, "helm upgrade succeeded")
			}
		} else {
			logger.Info("no action required. running status check")
			if resources, err := GetHelmManifest(ctx, res.Namespace, *res.Deployment); err != nil {
				logger.Warn("could not get helm manifest", zap.Error(err))
				pushErrorStatus(ctx, fmt.Errorf("could not get helm manifest: %w", err))
			} else {
				var err error
				for _, resource := range resources {
					logger.Sugar().Debugf("check status for %v %v", resource.GetKind(), resource.GetName())
					if err = CheckStatus(ctx, res.Namespace, resource); err != nil {
						break
					}
				}
				if err != nil {
					logger.Warn("resource status error", zap.Error(err))
					pushErrorStatus(ctx, fmt.Errorf("resource status error: %w", err))
				} else {
					logger.Info("status check passed")
					pushStatus(ctx, fmt.Sprintf("status check passed. %v resources", len(resources)))
				}
			}
		}

	}

	logger.Info("shutting down")
}
