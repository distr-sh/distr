package main

import (
	"context"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
)

func ForceRestart(ctx context.Context, namespace string, d ControllerDeployment) error {
	logger := logger.With(zap.Any("deploymentId", d.ID))
	logger.Info("performing force restart")
	manifest, err := GetHelmManifest(ctx, namespace, d.ReleaseName)
	if err != nil {
		return fmt.Errorf("could not get helm manifest: %w", err)
	}

	var aggregateErr error

	for _, obj := range FromUnstructuredSlice(manifest) {
		gvk := obj.GetObjectKind().GroupVersionKind()
		logger := logger.With(zap.String("resourceKind", gvk.Kind))
		metaObj, ok := obj.(metav1.Object)
		if !ok {
			logger.Warn("skipping non-metav1 object", zap.Error(err))
			continue
		}
		logger = logger.With(zap.String("resourceName", metaObj.GetName()))

		before, err := runtime.Encode(scheme.DefaultJSONEncoder(), obj)
		if err != nil {
			multierr.AppendInto(&aggregateErr, fmt.Errorf(
				"failed to encode object %v with kind %v: %w",
				metaObj.GetName(), gvk.Kind, err,
			))
			continue
		}

		after, err := polymorphichelpers.ObjectRestarterFn(obj)
		if err != nil {
			logger.Warn("failed to apply ObjectRestarterFn", zap.Error(err))
			continue
		}
		if after == nil {
			logger.Debug("skipping empty result", zap.Error(err))
			continue
		}

		patch, err := strategicpatch.CreateTwoWayMergePatch(before, after, obj)
		if err != nil {
			multierr.AppendInto(&aggregateErr, fmt.Errorf(
				"failed to generate patch for object %v with kind %v: %w",
				metaObj.GetName(), gvk.Kind, err,
			))
			continue
		}

		mapping, err := k8sRestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			multierr.AppendInto(&aggregateErr, fmt.Errorf(
				"failed to create RESTMapping for object %v with kind %v: %w",
				metaObj.GetName(), gvk.Kind, err,
			))
			continue
		}

		var resource dynamic.ResourceInterface
		if mapping.Scope == meta.RESTScopeNamespace {
			resource = k8sDynamicClient.Resource(mapping.Resource).Namespace(namespace)
		} else {
			resource = k8sDynamicClient.Resource(mapping.Resource)
		}

		_, err = resource.Patch(
			ctx,
			metaObj.GetName(),
			types.StrategicMergePatchType,
			patch,
			metav1.PatchOptions{
				FieldManager: fieldManager,
			},
		)
		if err != nil {
			multierr.AppendInto(&aggregateErr, fmt.Errorf(
				"failed to apply patch for object %v with kind %v: %w",
				gvk.Kind, metaObj.GetName(), err,
			))
			continue
		}
	}

	return aggregateErr
}
