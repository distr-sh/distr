package util

import (
	"errors"
	"fmt"
	"iter"
	"strings"
)

type mergeError struct {
	key     string
	wrapped error
}

func (err *mergeError) Error() string {
	path := []string{err.key}
	var root error
	next := err.wrapped
	for {
		if nm, ok := next.(*mergeError); ok {
			path = append(path, nm.key)
			next = nm.wrapped
		} else {
			root = next
			break
		}
	}
	return fmt.Sprintf("merge error at %v: %v", strings.Join(path, "."), root)
}

func MergeAllRecursive(mm ...map[string]any) (map[string]any, error) {
	dst := map[string]any{}
	for _, src := range mm {
		if err := MergeIntoRecursive(dst, src); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

func MergeIntoRecursive(dst, src map[string]any) error {
	for k, sv := range src {
		if svm, ok := sv.(map[string]any); ok {
			if _, ok := dst[k].(map[string]any); !ok && dst[k] != nil {
				return &mergeError{key: k, wrapped: errors.New("can not merge map into non-map type")}
			}
			if dst[k] == nil {
				dst[k] = map[string]any{}
			}
			if err := MergeIntoRecursive(dst[k].(map[string]any), svm); err != nil {
				return &mergeError{key: k, wrapped: err}
			}
		} else if _, ok := dst[k].(map[string]any); ok {
			return &mergeError{key: k, wrapped: errors.New("can not merge non-map type into map")}
		} else {
			dst[k] = sv
		}
	}
	return nil
}

func GetValues[K comparable, V any](src map[K]V) []V {
	values := make([]V, 0, len(src))
	for _, value := range src {
		values = append(values, value)
	}
	return values
}

func CollectKeys[K comparable](src iter.Seq[K]) map[K]struct{} {
	dst := make(map[K]struct{})
	InsertKeys(dst, src)
	return dst
}

func InsertKeys[K comparable, T any](dst map[K]T, src iter.Seq[K]) {
	var t T
	for v := range src {
		if _, ok := dst[v]; !ok {
			dst[v] = t
		}
	}
}
