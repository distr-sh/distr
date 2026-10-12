package main

import (
	"bytes"
	"context"
	"errors"
	"io"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
)

func DecodeResourceYaml(data []byte) ([]*unstructured.Unstructured, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(data), 4096)
	var result []*unstructured.Unstructured
	for {
		var object unstructured.Unstructured
		if err := decoder.Decode(&object); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		} else if len(object.Object) > 0 {
			result = append(result, &object)
		}
	}
	return result, nil
}

func FromUnstructuredSlice(src []*unstructured.Unstructured) []runtime.Object {
	result := make([]runtime.Object, len(src))
	for i, obj := range src {
		if converted, err := FromUnstructured(obj); err != nil {
			logger.With(zap.Error(err)).Sugar().Warn("cannot convert unstructured with kind %v", obj.GetKind())
			result[i] = obj
		} else {
			result[i] = converted
		}
	}
	return result
}

func FromUnstructured(src *unstructured.Unstructured) (runtime.Object, error) {
	if dst, err := scheme.Scheme.New(src.GroupVersionKind()); err != nil {
		return nil, err
	} else if err := runtime.DefaultUnstructuredConverter.FromUnstructured(src.Object, dst); err != nil {
		return nil, err
	} else {
		return dst, nil
	}
}

func ApplyResources(ctx context.Context, namespace string, objects []*unstructured.Unstructured) error {
	for _, obj := range objects {
		gvk := obj.GroupVersionKind()
		logger := logger.With(
			zap.String("resourceNamespace", namespace),
			zap.String("resourceKind", gvk.Kind),
			zap.String("resourceName", obj.GetName()),
		)
		if mapping, err := k8sRestMapper.RESTMapping(gvk.GroupKind(), gvk.Version); err != nil {
			return err
		} else {
			var resource dynamic.ResourceInterface
			if mapping.Scope.Name() != meta.RESTScopeNameRoot {
				resource = k8sDynamicClient.Resource(mapping.Resource).Namespace(namespace)
			} else {
				resource = k8sDynamicClient.Resource(mapping.Resource)
			}
			if _, err := resource.Get(ctx, obj.GetName(), v1.GetOptions{}); k8serrors.IsNotFound(err) {
				logger.Debug("creating resource")
				if _, err := resource.Create(ctx, obj, v1.CreateOptions{}); err != nil {
					return err
				}
			} else if err == nil {
				logger.Debug("updating resource")
				if _, err := resource.Update(ctx, obj, v1.UpdateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}
