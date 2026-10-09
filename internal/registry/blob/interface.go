package blob

import (
	"context"
	"io"

	"github.com/opencontainers/go-digest"
)

// BlobHandler represents a minimal blob storage backend, capable of serving
// blob contents.
type BlobHandler interface {
	// Get gets the blob contents, or errNotFound if the blob wasn't found.
	Get(ctx context.Context, repo string, h digest.Digest, allowRedirect bool) (io.ReadCloser, error)
}

// BlobStatHandler is an extension interface representing a blob storage
// backend that can serve metadata about blobs.
type BlobStatHandler interface {
	// Stat returns the size of the blob, or errNotFound if the blob wasn't
	// found, or redirectError if the blob can be found elsewhere.
	Stat(ctx context.Context, repo string, h digest.Digest) (int64, error)
}

// BlobPutHandler is an extension interface representing a blob storage backend
// that can write blob contents.
type BlobPutHandler interface {
	// Put puts the blob contents.
	//
	// The contents will be verified against the expected size and digest
	// as the contents are read, and an error will be returned if these
	// don't match. Implementations should return that error, or a wrapper
	// around that error, to return the correct error when these don't match.
	Put(ctx context.Context, repo string, h digest.Digest, contentType string, r io.Reader) error
	StartSession(ctx context.Context, repo string) (string, error)
	CompleteSession(ctx context.Context, repo, id string, digest digest.Digest) error
	PutChunk(ctx context.Context, id string, r io.Reader, start int64) (int64, error)
	GetUploadedPartsSize(ctx context.Context, id string) (int64, error)
}

// BlobChunkMinLengthHandler is an extension interface representing a blob storage backend that
// requires every chunk of a chunked upload except the last to have a minimum size.
type BlobChunkMinLengthHandler interface {
	ChunkMinLength() int64
}

// BlobDeleteHandler is an extension interface representing a blob storage
// backend that can delete blob contents.
type BlobDeleteHandler interface {
	// Delete the blob contents.
	Delete(ctx context.Context, repo string, h digest.Digest) error
}
