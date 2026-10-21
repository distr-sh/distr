// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/registry/authz"
	"github.com/distr-sh/distr/internal/registry/blob"
	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/registry/verify"
	"github.com/glasskube/pkg/seekbuf"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"go.uber.org/zap"
)

// UpstreamBlobFetcher fetches a blob from an upstream registry.
type UpstreamBlobFetcher interface {
	// OpenBlob returns the size and the content of the blob identified by d at the upstream registry for repo.
	// The content ends with an error if it does not match d. The caller must close it.
	// Returns distrerr.ErrNotFound if repo has no upstream configured.
	OpenBlob(ctx context.Context, repo string, d digest.Digest) (int64, io.ReadCloser, error)
}

const (
	uploads = "uploads"
)

// Returns whether this url should be handled by the blob handler
// This is complicated because blob is indicated by the trailing path, not the leading path.
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#pulling-a-layer
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#pushing-a-layer
func isBlob(req *http.Request) bool {
	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	if elem[len(elem)-1] == "" {
		elem = elem[:len(elem)-1]
	}
	if len(elem) < 3 {
		return false
	}
	return elem[len(elem)-2] == "blobs" || (elem[len(elem)-3] == "blobs" &&
		elem[len(elem)-2] == uploads)
}

// blobs
type blobs struct {
	blobHandler     blob.BlobHandler
	authz           authz.Authorizer
	log             *zap.SugaredLogger
	upstreamFetcher UpstreamBlobFetcher
}

func (b *blobs) handle(resp http.ResponseWriter, req *http.Request) *regError {
	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	if elem[len(elem)-1] == "" {
		elem = elem[:len(elem)-1]
	}
	// Must have a path of form /v2/{name}/blobs/{upload,sha256:}
	if len(elem) < 4 {
		return &regError{
			Status:  http.StatusBadRequest,
			Code:    "NAME_INVALID",
			Message: "blobs must be attached to a repo",
		}
	}
	target := elem[len(elem)-1]
	service := elem[len(elem)-2]
	digestFromQuery := req.URL.Query().Get("digest")
	contentRange := req.Header.Get("Content-Range")
	rangeHeader := req.Header.Get("Range")
	repoEnd := len(elem) - 2
	if service == uploads {
		repoEnd--
	}
	repo := req.URL.Host + path.Join(elem[1:repoEnd]...)

	switch req.Method {
	case http.MethodHead:
		if h, err := digest.Parse(target); err != nil {
			return regErrDigestInvalid
		} else if err := b.authz.AuthorizeBlob(req.Context(), repo, h, authz.ActionStat); err != nil {
			return regErrBlobAuthz(err)
		}
		return b.handleHead(resp, req, repo, target)
	case http.MethodGet:
		if h, err := digest.Parse(target); err == nil {
			if err := b.authz.AuthorizeBlob(req.Context(), repo, h, authz.ActionRead); err != nil {
				return regErrBlobAuthz(err)
			}
		} else if _, err := uuid.Parse(target); err != nil {
			return regErrDigestInvalid
		} else if err := b.authz.Authorize(req.Context(), repo, authz.ActionWrite); err != nil {
			// Reading an upload by its id belongs to a push, so it takes the same authorization.
			return regErrAuthz(err)
		}
		return b.handleGet(resp, req, repo, target, rangeHeader)
	case http.MethodPost:
		if err := b.authz.Authorize(req.Context(), repo, authz.ActionWrite); err != nil {
			return regErrAuthz(err)
		}
		return b.handlePost(resp, req, repo, target, digestFromQuery)
	case http.MethodPatch:
		if err := b.authz.Authorize(req.Context(), repo, authz.ActionWrite); err != nil {
			return regErrAuthz(err)
		}
		return b.handlePatch(resp, req, target, service, contentRange)
	case http.MethodPut:
		if h, err := digest.Parse(digestFromQuery); err != nil {
			return regErrDigestInvalid
		} else if err := b.authz.AuthorizeBlob(req.Context(), repo, h, authz.ActionWrite); err != nil {
			return regErrBlobAuthz(err)
		}
		return b.handlePut(resp, req, service, repo, target, digestFromQuery, contentRange)
	case http.MethodDelete:
		// Blobs are shared between artifacts and organizations and are removed by the ArtifactBlob cleanup.
		return regErrUnsupported
	default:
		return regErrMethodUnknown
	}
}

func (b *blobs) handleHead(resp http.ResponseWriter, req *http.Request, repo, target string) *regError {
	h, err := digest.Parse(target)
	if err != nil {
		return regErrDigestInvalid
	}

	size, err := b.blobHandler.Stat(req.Context(), repo, h)
	if errors.Is(err, blob.ErrNotFound) {
		if s, upstreamErr := b.statFromUpstreamDB(req.Context(), repo, h); upstreamErr == nil {
			size = s
		} else {
			return regErrBlobUnknown
		}
	} else if err != nil {
		var rerr blob.RedirectError
		if errors.As(err, &rerr) {
			http.Redirect(resp, req, rerr.Location, rerr.Code)
			return nil
		}
		return regErrInternal(err)
	}

	resp.Header().Set("Content-Length", fmt.Sprint(size))
	resp.Header().Set("Docker-Content-Digest", h.String())
	resp.WriteHeader(http.StatusOK)
	return nil
}

// statFromUpstreamDB returns the blob size from the DB for pull-through artifacts.
// The size is available because manifest sync eagerly stores all blob references with sizes.
func (b *blobs) statFromUpstreamDB(ctx context.Context, repo string, h digest.Digest) (int64, error) {
	n, err := name.Parse(repo)
	if err != nil {
		return 0, distrerr.ErrNotFound
	}
	return db.GetPullThroughBlobSize(ctx, n.OrgName, n.ArtifactName, h.String())
}

func (b *blobs) handleGet(resp http.ResponseWriter, req *http.Request, repo, target, rangeHeader string) *regError {
	h, err := digest.Parse(target)
	if err != nil {
		if id, err := uuid.Parse(target); err != nil {
			return regErrDigestInvalid
		} else if bph, ok := b.blobHandler.(blob.BlobPutHandler); !ok {
			return regErrUnsupported
		} else if uploaded, err := bph.GetUploadedPartsSize(req.Context(), id.String()); err != nil {
			return regErrInternal(err)
		} else {
			resp.Header().Set("Location", "/"+path.Join("v2", repo, "blobs/uploads", target))
			resp.Header().Set("Range", fmt.Sprintf("0-%v", max(uploaded-1, 0)))
			resp.WriteHeader(http.StatusNoContent)
			return nil
		}
	}

	size, err := b.blobHandler.Stat(req.Context(), repo, h)
	if errors.Is(err, blob.ErrNotFound) {
		return b.serveFromUpstream(resp, req, repo, h)
	} else if err != nil {
		var rerr blob.RedirectError
		if errors.As(err, &rerr) {
			http.Redirect(resp, req, rerr.Location, rerr.Code)
			return nil
		}
		return regErrInternal(err)
	}

	rc, err := b.blobHandler.Get(req.Context(), repo, h, true)
	if errors.Is(err, blob.ErrNotFound) {
		return regErrBlobUnknown
	} else if err != nil {
		var rerr blob.RedirectError
		if errors.As(err, &rerr) {
			http.Redirect(resp, req, rerr.Location, rerr.Code)
			return nil
		}
		return regErrInternal(err)
	}
	defer rc.Close()
	var r io.Reader = rc

	if rangeHeader != "" {
		start, end := int64(0), int64(0)
		if _, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); err != nil {
			return &regError{
				Status:  http.StatusRequestedRangeNotSatisfiable,
				Code:    "BLOB_UNKNOWN",
				Message: "We don't understand your Range",
			}
		}

		n := (end + 1) - start
		if ra, ok := r.(io.ReaderAt); ok {
			if end+1 > size {
				return &regError{
					Status:  http.StatusRequestedRangeNotSatisfiable,
					Code:    "BLOB_UNKNOWN",
					Message: fmt.Sprintf("range end %d > %d size", end+1, size),
				}
			}
			r = io.NewSectionReader(ra, start, n)
		} else {
			if _, err := io.CopyN(io.Discard, r, start); err != nil {
				return &regError{
					Status:  http.StatusRequestedRangeNotSatisfiable,
					Code:    "BLOB_UNKNOWN",
					Message: fmt.Sprintf("Failed to discard %d bytes", start),
				}
			}

			r = io.LimitReader(r, n)
		}

		resp.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
		resp.Header().Set("Content-Length", fmt.Sprint(n))
		resp.Header().Set("Docker-Content-Digest", h.String())
		resp.WriteHeader(http.StatusPartialContent)
	} else {
		resp.Header().Set("Content-Length", fmt.Sprint(size))
		resp.Header().Set("Docker-Content-Digest", h.String())
		resp.WriteHeader(http.StatusOK)
	}

	if _, err := io.Copy(resp, r); err != nil {
		return regErrInternal(err)
	}
	return nil
}

// serveFromUpstream streams a blob of a pull-through artifact from its upstream registry to the client and stores
// it once the download is complete. The response has to start before the download finishes, since proxies in front
// of the registry give up on a response that has not started after a fixed time, commonly 30 seconds. The download
// outlives the request so that the retry of a client that gave up finds the blob stored.
func (b *blobs) serveFromUpstream(resp http.ResponseWriter, req *http.Request, repo string, h digest.Digest) *regError {
	if b.upstreamFetcher == nil {
		return regErrBlobUnknown
	}
	bph, ok := b.blobHandler.(blob.BlobPutHandler)
	if !ok {
		return regErrBlobUnknown
	}

	log := internalctx.GetLogger(req.Context()).With(zap.String("digest", h.String()))
	ctx, cancel := context.WithTimeout(context.WithoutCancel(req.Context()), env.RegistryUpstreamBlobFetchTimeout())
	ctx = internalctx.WithLogger(ctx, log)
	size, rc, err := b.upstreamFetcher.OpenBlob(ctx, repo, h)
	if err != nil {
		cancel()
		if !errors.Is(err, distrerr.ErrNotFound) {
			log.Warn("upstream blob fetch failed", zap.Error(err))
		}
		return regErrBlobUnknown
	}

	pr, pw := io.Pipe()
	copied := make(chan struct{})
	go func() {
		defer cancel()
		storeUpstreamBlob(ctx, repo, h, bph, pr)
		<-copied
	}()

	resp.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	resp.Header().Set("Docker-Content-Digest", h.String())
	resp.WriteHeader(http.StatusOK)

	w := &upstreamBlobWriter{client: resp, buffer: pw}
	_, err = io.Copy(w, rc)
	_ = rc.Close()
	_ = pw.CloseWithError(err)
	close(copied)

	// Unless buffering failed before, storeUpstreamBlob receives err through the pipe and logs it.
	if err != nil && w.bufferErr != nil && !errors.Is(err, errUpstreamBlobAbandoned) {
		log.Warn("upstream blob fetch failed", zap.Error(err))
	}
	return nil
}

func storeUpstreamBlob(ctx context.Context, repo string, h digest.Digest, bph blob.BlobPutHandler, r *io.PipeReader) {
	log := internalctx.GetLogger(ctx)

	buf, err := seekbuf.New(r)
	if err != nil {
		_ = r.CloseWithError(err)
		log.Warn("failed to buffer upstream blob", zap.Error(err))
		return
	}
	defer func() {
		if err := buf.Destroy(); err != nil {
			log.Warn("ephemeral resource cleanup error", zap.Error(err))
		}
	}()

	sr, err := buf.Get()
	if err != nil {
		log.Warn("failed to open buffered upstream blob", zap.Error(err))
		return
	}
	defer sr.Close()

	if err := bph.Put(ctx, repo, h, "", sr); err != nil {
		log.Warn("failed to store upstream blob", zap.Error(err))
	}
}

var errUpstreamBlobAbandoned = errors.New("neither the client nor the buffer accepts the upstream blob")

// upstreamBlobWriter writes an upstream blob to the client and to the buffer, and keeps writing to one of them after
// the other failed.
type upstreamBlobWriter struct {
	client, buffer       io.Writer
	clientErr, bufferErr error
}

func (w *upstreamBlobWriter) Write(p []byte) (int, error) {
	if w.clientErr == nil {
		_, w.clientErr = w.client.Write(p)
	}
	if w.bufferErr == nil {
		_, w.bufferErr = w.buffer.Write(p)
	}
	if w.clientErr != nil && w.bufferErr != nil {
		return 0, errUpstreamBlobAbandoned
	}
	return len(p), nil
}

func (b *blobs) handlePost(resp http.ResponseWriter, req *http.Request, repo, target, digestArg string) *regError {
	bph, ok := b.blobHandler.(blob.BlobPutHandler)
	if !ok {
		return regErrUnsupported
	}

	// It is weird that this is "target" instead of "service", but
	// that's how the index math works out above.
	if target != uploads {
		return &regError{
			Status:  http.StatusBadRequest,
			Code:    "METHOD_UNKNOWN",
			Message: fmt.Sprintf("POST to /blobs must be followed by /uploads, got %s", target),
		}
	}

	if digestArg != "" {
		h, err := digest.Parse(digestArg)
		if err != nil {
			return regErrDigestInvalid
		}

		vrc, err := verify.ReadCloser(req.Body, req.ContentLength, h)
		if err != nil {
			return regErrInternal(err)
		}
		defer vrc.Close()

		if err = bph.Put(req.Context(), repo, h, "", vrc); err != nil {
			if _, ok := errors.AsType[verify.Error](err); ok {
				log := internalctx.GetLogger(req.Context())
				log.Warn("Digest mismatch detected", zap.Error(err))
				return regErrDigestMismatch
			}
			return regErrInternal(err)
		}
		resp.Header().Set("Docker-Content-Digest", h.String())
		resp.Header().Set("Location", req.URL.JoinPath("..", h.String()).Path)
		resp.WriteHeader(http.StatusCreated)
		return nil
	}

	if id, err := bph.StartSession(req.Context(), repo); err != nil {
		return regErrInternal(err)
	} else {
		if cmh, ok := b.blobHandler.(blob.BlobChunkMinLengthHandler); ok {
			resp.Header().Set("OCI-Chunk-Min-Length", strconv.FormatInt(cmh.ChunkMinLength(), 10))
		}
		resp.Header().Set("Location", req.URL.JoinPath(id).Path)
		resp.Header().Set("Range", "0-0")
		resp.WriteHeader(http.StatusAccepted)
		return nil
	}
}

func (b *blobs) handlePatch(
	resp http.ResponseWriter,
	req *http.Request,
	target, service, contentRange string,
) *regError {
	bph, ok := b.blobHandler.(blob.BlobPutHandler)
	if !ok {
		return regErrUnsupported
	}

	if service != uploads {
		return &regError{
			Status:  http.StatusBadRequest,
			Code:    "METHOD_UNKNOWN",
			Message: fmt.Sprintf("PATCH to /blobs must be followed by /uploads, got %s", service),
		}
	}

	var start, end int64 = 0, 0
	if contentRange != "" {
		if _, err := fmt.Sscanf(contentRange, "%d-%d", &start, &end); err != nil {
			return &regError{
				Status:  http.StatusRequestedRangeNotSatisfiable,
				Code:    "BLOB_UPLOAD_UNKNOWN",
				Message: "We don't understand your Content-Range",
			}
		}
	}

	size, err := bph.PutChunk(req.Context(), target, req.Body, start)
	if errors.Is(err, blob.ErrBadUpload) {
		return &regError{
			Status:  http.StatusRequestedRangeNotSatisfiable,
			Code:    "BLOB_UPLOAD_INVALID",
			Message: err.Error(),
		}
	} else if err != nil {
		return regErrInternal(err)
	}

	resp.Header().Set("Location", req.URL.Path)
	resp.Header().Set("Range", fmt.Sprintf("0-%d", size-1))
	resp.WriteHeader(http.StatusAccepted)
	return nil
}

func (b *blobs) handlePut(
	resp http.ResponseWriter,
	req *http.Request,
	service, repo, target, digestArg, contentRange string,
) *regError {
	bph, ok := b.blobHandler.(blob.BlobPutHandler)
	if !ok {
		return regErrUnsupported
	}

	if service != uploads {
		return &regError{
			Status:  http.StatusBadRequest,
			Code:    "METHOD_UNKNOWN",
			Message: fmt.Sprintf("PUT to /blobs must be followed by /uploads, got %s", service),
		}
	}

	if digestArg == "" {
		return &regError{
			Status:  http.StatusBadRequest,
			Code:    "DIGEST_INVALID",
			Message: "digest not specified",
		}
	}

	h, err := digest.Parse(digestArg)
	if err != nil {
		return regErrDigestInvalid
	}

	if req.ContentLength > 0 {
		var start, end int64 = 0, 0
		if contentRange != "" {
			if _, err := fmt.Sscanf(contentRange, "%d-%d", &start, &end); err != nil {
				return &regError{
					Status:  http.StatusRequestedRangeNotSatisfiable,
					Code:    "BLOB_UPLOAD_UNKNOWN",
					Message: "We don't understand your Content-Range",
				}
			}
		}
		size, err := bph.PutChunk(req.Context(), target, req.Body, start)
		if errors.Is(err, blob.ErrBadUpload) {
			return &regError{
				Status:  http.StatusRequestedRangeNotSatisfiable,
				Code:    "BLOB_UPLOAD_INVALID",
				Message: err.Error(),
			}
		} else if err != nil {
			return regErrInternal(err)
		} else if contentRange != "" && size != end+1 {
			return &regError{
				Status:  http.StatusRequestedRangeNotSatisfiable,
				Code:    "BLOB_UPLOAD_INVALID",
				Message: "size of uploaded chunks does not match requested range",
			}
		}
	}

	err = bph.CompleteSession(req.Context(), repo, target, h)
	if errors.Is(err, blob.ErrBadUpload) && h == h.Algorithm().FromBytes(nil) {
		// A session that never received a chunk has no upload to complete.
		err = bph.Put(req.Context(), repo, h, "", bytes.NewReader(nil))
	}
	if _, ok := errors.AsType[verify.Error](err); ok {
		internalctx.GetLogger(req.Context()).Warn("Digest mismatch detected", zap.Error(err))
		return regErrDigestMismatch
	} else if errors.Is(err, blob.ErrBadUpload) {
		return regErrBlobUploadUnknown
	} else if err != nil {
		return regErrInternal(err)
	}

	resp.Header().Set("Docker-Content-Digest", h.String())
	resp.Header().Set("Location", "/"+path.Join("v2", repo, "blobs", h.String()))
	resp.WriteHeader(http.StatusCreated)
	return nil
}
