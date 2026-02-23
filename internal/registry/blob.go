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
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/registry/authz"
	"github.com/distr-sh/distr/internal/registry/blob"
	registryerror "github.com/distr-sh/distr/internal/registry/error"
	"github.com/distr-sh/distr/internal/registry/verify"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"go.uber.org/zap"
)

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
	blobHandler blob.BlobHandler
	authz       authz.Authorizer
	log         *zap.SugaredLogger
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
	repo := req.URL.Host + path.Join(elem[1:len(elem)-2]...)

	switch req.Method {
	case http.MethodHead:
		if h, err := digest.Parse(target); err != nil {
			return regErrDigestInvalid
		} else if err := b.authz.AuthorizeBlob(req.Context(), h, authz.ActionStat); err != nil {
			if errors.Is(err, authz.ErrAccessDenied) {
				return regErrDenied(err.Error())
			} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
				return regErrNameInvalid
			}
			return regErrInternal(err)
		}
		return b.handleHead(resp, req, repo, target)
	case http.MethodGet:
		if h, err := digest.Parse(target); err == nil {
			if err := b.authz.AuthorizeBlob(req.Context(), h, authz.ActionRead); err != nil {
				if errors.Is(err, authz.ErrAccessDenied) {
					return regErrDenied(err.Error())
				} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
					return regErrNameInvalid
				}
				return regErrInternal(err)
			}
		} else if _, err := uuid.Parse(target); err != nil {
			return regErrDigestInvalid
		}
		return b.handleGet(resp, req, repo, target, rangeHeader)
	case http.MethodPost:
		if err := b.authz.Authorize(req.Context(), repo, authz.ActionWrite); err != nil {
			if errors.Is(err, authz.ErrAccessDenied) {
				return regErrDenied(err.Error())
			} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
				return regErrNameInvalid
			}
			return regErrInternal(err)
		}
		return b.handlePost(resp, req, repo, target, digestFromQuery)
	case http.MethodPatch:
		if err := b.authz.Authorize(req.Context(), repo, authz.ActionWrite); err != nil {
			if errors.Is(err, authz.ErrAccessDenied) {
				return regErrDenied(err.Error())
			} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
				return regErrNameInvalid
			}
			return regErrInternal(err)
		}
		return b.handlePatch(resp, req, target, service, contentRange)
	case http.MethodPut:
		if h, err := digest.Parse(digestFromQuery); err != nil {
			return regErrDigestInvalid
		} else if err := b.authz.AuthorizeBlob(req.Context(), h, authz.ActionWrite); err != nil {
			if errors.Is(err, authz.ErrAccessDenied) {
				return regErrDenied(err.Error())
			} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
				return regErrNameInvalid
			}
			return regErrInternal(err)
		}
		return b.handlePut(resp, req, service, repo, target, digestFromQuery, contentRange)
	// case http.MethodDelete:
	// 	if err := b.authz.AuthorizeBlob(req.Context(), targetHash, authz.ActionWrite); err != nil {
	// 		if errors.Is(err, authz.ErrAccessDenied) {
	// 			return regErrDenied
	// 		}
	// 		return regErrInternal(err)
	// 	}
	// 	return b.handleDelete(resp, req, repo, target)
	default:
		return regErrMethodUnknown
	}
}

func (b *blobs) handleHead(resp http.ResponseWriter, req *http.Request, repo, target string) *regError {
	h, err := digest.Parse(target)
	if err != nil {
		return regErrDigestInvalid
	}

	var size int64
	if bsh, ok := b.blobHandler.(blob.BlobStatHandler); ok {
		size, err = bsh.Stat(req.Context(), repo, h)
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
	} else {
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
		size, err = io.Copy(io.Discard, rc)
		if err != nil {
			return regErrInternal(err)
		}
	}

	resp.Header().Set("Content-Length", fmt.Sprint(size))
	resp.Header().Set("Docker-Content-Digest", h.String())
	resp.WriteHeader(http.StatusOK)
	return nil
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
			resp.Header().Set("Range", fmt.Sprintf("0-%v", uploaded-1))
			resp.WriteHeader(http.StatusNoContent)
			return nil
		}
	}

	var size int64
	var r io.Reader
	if bsh, ok := b.blobHandler.(blob.BlobStatHandler); ok {
		size, err = bsh.Stat(req.Context(), repo, h)
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
		r = rc
	} else {
		tmp, err := b.blobHandler.Get(req.Context(), repo, h, true)
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
		defer tmp.Close()
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, tmp); err != nil {
			return regErrInternal(err)
		}
		size = int64(buf.Len())
		r = &buf
	}

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
			if errors.As(err, &verify.Error{}) {
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
		} else if contentRange != "" && size != end {
			return &regError{
				Status:  http.StatusRequestedRangeNotSatisfiable,
				Code:    "BLOB_UPLOAD_INVALID",
				Message: "size of uploaded chunks does not match requested range",
			}
		}
	}

	err = bph.CompleteSession(req.Context(), repo, target, h)
	if err != nil {
		return regErrInternal(err)
	}

	resp.Header().Set("Docker-Content-Digest", h.String())
	resp.Header().Set("Location", req.URL.JoinPath("..", h.String()).Path)
	resp.WriteHeader(http.StatusCreated)
	return nil
}

// func (b *blobs) handleDelete(resp http.ResponseWriter, req *http.Request, repo, target string) *regError {
// 	bdh, ok := b.blobHandler.(blob.BlobDeleteHandler)
// 	if !ok {
// 		return regErrUnsupported
// 	}

// 	h, err := v1.NewHash(target)
// 	if err != nil {
// 		return regErrDigestInvalid
// 	}
// 	if err := bdh.Delete(req.Context(), repo, h); err != nil {
// 		return regErrInternal(err)
// 	}
// 	resp.WriteHeader(http.StatusAccepted)
// 	return nil
// }
