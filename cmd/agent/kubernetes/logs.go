package main

import (
	"bufio"
	"context"
	"fmt"
	"maps"
	"sync/atomic"
	"time"

	"github.com/distr-sh/distr/internal/deploymentlogs"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	applyconfigurationscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/polymorphichelpers"
)

type logsWatcher struct {
	interval  time.Duration
	logsAfter atomic.Pointer[time.Time]
	namespace atomic.Value
}

func NewLogsWatcher(namespace string, interval time.Duration) *logsWatcher {
	lw := logsWatcher{interval: interval}
	lw.namespace.Store(namespace)
	return &lw
}

func (lw *logsWatcher) Watch(ctx context.Context) {
	logger.Debug("logs watcher is starting to watch",
		zap.String("namespace", lw.GetNamespace()),
		zap.Duration("interval", lw.interval))
	tick := time.Tick(lw.interval)
	for {
		lw.collect(ctx)
		select {
		case <-ctx.Done():
			logger.Debug("log watcher stopped", zap.Error(ctx.Err()))
			return
		case <-tick:
			continue
		}
	}
}

func (lw *logsWatcher) SetLogsAfter(v *time.Time) {
	lw.logsAfter.Store(v)
}

func (lw *logsWatcher) SetNamespace(ns string) {
	lw.namespace.Store(ns)
}

func (lw *logsWatcher) GetNamespace() string {
	if s, ok := lw.namespace.Load().(string); ok {
		return s
	} else {
		return ""
	}
}

func (lw *logsWatcher) collect(ctx context.Context) {
	namespace := lw.GetNamespace()

	logger.Debug("getting logs")
	existingDeployments, err := GetExistingDeployments(ctx, namespace)
	if err != nil {
		logger.Error("could not get existing deployments", zap.Error(err))
		return
	}

	collector := deploymentlogs.NewCollector(agentClient, logger)

	for _, d := range existingDeployments {
		logger := logger.With(zap.Any("deploymentId", d.ID))
		sinceTime, err := lw.GetLastLogsTimestamp(ctx, namespace, d)
		if err != nil {
			logger.Error("could not get last logs timestamp for deployment", zap.Error(err))
			continue
		}

		var resources []runtime.Object
		if resUnstr, err := GetHelmManifest(ctx, namespace, d.ReleaseName); err != nil {
			logger.Error("could not get helm manifest for deployment", zap.Error(err))
			continue
		} else {
			resources = FromUnstructuredSlice(resUnstr)
		}

		deploymentCollector := collector.For(d)
		now := time.Now()
		var toplevelErr error

		responseMap := map[corev1.ObjectReference]rest.ResponseWrapper{}
		resourceNameMap := map[string]string{}
		for _, obj := range resources {
			logger := logger.With(zap.String("resourceKind", obj.GetObjectKind().GroupVersionKind().Kind))

			var resourceName string
			if metaObj, ok := obj.(metav1.Object); ok {
				metaObj.SetNamespace(namespace)
				logger = logger.With(zap.String("resourceName", metaObj.GetName()))

				if restMapping, err := k8sRestMapper.RESTMapping(obj.GetObjectKind().GroupVersionKind().GroupKind()); err != nil {
					logger.Warn("could not get REST mapping for resource", zap.Error(err))
					toplevelErr = err
					break
				} else {
					resourceName = fmt.Sprintf("%v/%v", restMapping.Resource.Resource, metaObj.GetName())
				}
			}

			logOptions := corev1.PodLogOptions{Timestamps: true}
			if sinceTime != nil {
				logOptions.SinceTime = &metav1.Time{Time: *sinceTime}
			}

			logger.Debug("get logs for resource", zap.Timep("sinceTime", sinceTime))

			resourceResponseMap, err := polymorphichelpers.AllPodLogsForObjectFn(
				k8sConfigFlags, obj, &logOptions, 10*time.Second, true,
			)
			if err != nil {
				// not being able to get logs for all resource types is normal so we only want to call abort when an
				// API error is encountered.
				if _, ok := err.(errors.APIStatus); ok {
					logger.Warn("could not get logs", zap.Error(err))
					toplevelErr = err
					break
				} else {
					logger.Debug("could not get logs", zap.Error(err))
				}
			} else {
				maps.Copy(responseMap, resourceResponseMap)
				for resource := range resourceResponseMap {
					resourceNameMap[resource.Name] = resourceName
				}
			}
		}

		for ref, resp := range responseMap {
			resourceName := resourceNameMap[ref.Name]
			if resourceName == "" {
				// fall back to pod name if no parent resource is available
				resourceName = ref.Name
			}

			err := func() error {
				rc, err := resp.Stream(ctx)
				if err != nil {
					logger.Warn("could not get logs for pod", zap.Error(err))
					return err
				}
				defer rc.Close()
				sc := bufio.NewScanner(rc)
				for sc.Scan() {
					if err := deploymentCollector.AppendMessage(ctx, resourceName, "Log", sc.Text()); err != nil {
						logger.Warn("error collecting log message", zap.Error(err))
						return err
					}
				}
				if err := sc.Err(); err != nil {
					logger.Warn("error streaming logs", zap.Error(err))
					return err
				}
				return nil
			}()
			if err != nil {
				toplevelErr = err
				break
			}
		}

		if toplevelErr == nil {
			if err := lw.UpdateLastLogsTimestamp(ctx, namespace, d, now); err != nil {
				logger.Warn("could not update last logs timestamp for deployment", zap.Error(err))
			}
		}
	}

	if err := collector.Flush(ctx); err != nil {
		logger.Warn("error exporting logs", zap.Error(err))
	}
}

func (lw *logsWatcher) UpdateLastLogsTimestamp(
	ctx context.Context,
	namespace string,
	deployment AgentDeployment,
	timestamp time.Time,
) error {
	_, err := k8sClient.CoreV1().Secrets(namespace).Apply(
		ctx,
		applyconfigurationscorev1.Secret(deployment.SecretName(), namespace).
			WithStringData(map[string]string{
				"lastLogTimestamp": timestamp.Format(time.RFC3339Nano),
			}),
		metav1.ApplyOptions{Force: true, FieldManager: "distr-agent-logs"},
	)
	return err
}

func (lw *logsWatcher) GetLastLogsTimestamp(
	ctx context.Context,
	namespace string,
	deployment AgentDeployment,
) (*time.Time, error) {
	logsAfter := lw.logsAfter.Load()

	if secret, err := k8sClient.CoreV1().Secrets(namespace).
		Get(ctx, deployment.SecretName(), metav1.GetOptions{}); err != nil {
		return nil, err
	} else if timeStr, ok := secret.Data["lastLogTimestamp"]; !ok {
		return logsAfter, nil
	} else if parsed, err := time.Parse(time.RFC3339Nano, string(timeStr)); err != nil {
		return nil, err
	} else if logsAfter != nil && parsed.Before(*logsAfter) {
		return logsAfter, nil
	} else {
		return &parsed, nil
	}
}
