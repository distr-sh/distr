package manifest

import (
	"context"

	"github.com/opencontainers/go-digest"
)

type ManifestHandler interface {
	List(ctx context.Context, n int) ([]string, error)
	// ListTags
	//
	// Spec for implementation:
	//
	// name: Name of the target repository.
	//
	// n: Limit the number of entries in each response. If not present, all entries will be returned.
	//
	// last: Result set will include values lexically after last.
	ListTags(ctx context.Context, name string, n int, last string) ([]string, error)
	ListDigests(ctx context.Context, name string) ([]digest.Digest, error)
	Get(ctx context.Context, name string, reference string) (*Manifest, error)
	Put(ctx context.Context, name string, reference string, manifest Manifest, blobs []Blob) error
	// Delete reports whether deleting the reference deleted the whole repository with it.
	Delete(ctx context.Context, name string, reference string) (bool, error)
}
