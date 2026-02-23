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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/containers/image/v5/manifest"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/registry/audit"
	"github.com/distr-sh/distr/internal/registry/authz"
	"github.com/distr-sh/distr/internal/registry/blob"
	registryerror "github.com/distr-sh/distr/internal/registry/error"
	imanifest "github.com/distr-sh/distr/internal/registry/manifest"
	"github.com/getsentry/sentry-go"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type catalog struct {
	Repos []string `json:"repositories"`
}

type listTags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type manifests struct {
	blobHandler     blob.BlobHandler
	manifestHandler imanifest.ManifestHandler
	authz           authz.Authorizer
	audit           audit.ArtifactAuditor
	log             *zap.SugaredLogger
}

func isManifest(req *http.Request) bool {
	elems := strings.Split(req.URL.Path, "/")
	elems = elems[1:]
	if len(elems) < 4 {
		return false
	}
	return elems[len(elems)-2] == "manifests"
}

func isTags(req *http.Request) bool {
	elems := strings.Split(req.URL.Path, "/")
	elems = elems[1:]
	if len(elems) < 4 {
		return false
	}
	return elems[len(elems)-2] == "tags"
}

func isCatalog(req *http.Request) bool {
	elems := strings.Split(req.URL.Path, "/")
	elems = elems[1:]
	if len(elems) < 2 {
		return false
	}

	return elems[len(elems)-1] == "_catalog"
}

// Returns whether this url should be handled by the referrers handler
func isReferrers(req *http.Request) bool {
	elems := strings.Split(req.URL.Path, "/")
	elems = elems[1:]
	if len(elems) < 4 {
		return false
	}
	return elems[len(elems)-2] == "referrers"
}

// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#pulling-an-image-manifest
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#pushing-an-image
func (handler *manifests) handle(resp http.ResponseWriter, req *http.Request) *regError {
	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	target := elem[len(elem)-1]
	repo := strings.Join(elem[1:len(elem)-2], "/")

	switch req.Method {
	case http.MethodGet:
		if err := handler.authz.AuthorizeReference(req.Context(), repo, target, authz.ActionRead); err != nil {
			if errors.Is(err, authz.ErrAccessDenied) {
				return regErrDenied(err.Error())
			} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
				return regErrNameInvalid
			}
			return regErrInternal(err)
		}
		return handler.handleGet(resp, req, repo, target)
	case http.MethodHead:
		if err := handler.authz.AuthorizeReference(req.Context(), repo, target, authz.ActionStat); err != nil {
			if errors.Is(err, authz.ErrAccessDenied) {
				return regErrDenied(err.Error())
			} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
				return regErrNameInvalid
			}
			return regErrInternal(err)
		}
		return handler.handleHead(resp, req, repo, target)
	case http.MethodPut:
		if err := handler.authz.AuthorizeReference(req.Context(), repo, target, authz.ActionWrite); err != nil {
			if errors.Is(err, authz.ErrAccessDenied) {
				return regErrDenied(err.Error())
			} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
				return regErrNameInvalid
			}
			return regErrInternal(err)
		}
		return handler.handlePut(resp, req, repo, target)
	case http.MethodDelete:
		if err := handler.authz.AuthorizeReference(req.Context(), repo, target, authz.ActionWrite); err != nil {
			if errors.Is(err, authz.ErrAccessDenied) {
				return regErrDenied(err.Error())
			} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
				return regErrNameInvalid
			}
			return regErrInternal(err)
		}
		return handler.handleDelete(resp, req, repo, target)
	default:
		return regErrMethodUnknown
	}
}

func (m *manifests) handleTags(resp http.ResponseWriter, req *http.Request) *regError {
	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	repo := strings.Join(elem[1:len(elem)-2], "/")

	if req.Method == http.MethodGet {
		if err := m.authz.Authorize(req.Context(), repo, authz.ActionRead); err != nil {
			if errors.Is(err, authz.ErrAccessDenied) {
				return regErrDenied(err.Error())
			} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
				return regErrNameInvalid
			}
			return regErrInternal(err)
		}

		last := req.URL.Query().Get("last")
		n := 10000
		if ns := req.URL.Query().Get("n"); ns != "" {
			if parsed, err := strconv.Atoi(ns); err != nil {
				return &regError{
					Status:  http.StatusBadRequest,
					Code:    "BAD_REQUEST",
					Message: fmt.Sprintf("parsing n: %v", err),
				}
			} else {
				n = parsed
			}
		}

		references, err := m.manifestHandler.ListTags(req.Context(), repo, n, last)
		if errors.Is(err, imanifest.ErrNameUnknown) {
			return regErrNameUnknown
		} else if err != nil {
			return regErrInternal(err)
		}

		tagsToList := listTags{
			Name: repo,
			Tags: references,
		}

		msg, err := json.Marshal(tagsToList)
		if err != nil {
			return regErrInternal(err)
		}
		resp.Header().Set("Content-Length", strconv.Itoa(len(msg)))
		resp.WriteHeader(http.StatusOK)
		if _, err := io.Copy(resp, bytes.NewReader(msg)); err != nil {
			return regErrInternal(err)
		}
		return nil
	}

	return regErrMethodUnknown
}

func (m *manifests) handleCatalog(resp http.ResponseWriter, req *http.Request) *regError {
	query := req.URL.Query()
	nStr := query.Get("n")
	n := 10000
	if nStr != "" {
		n, _ = strconv.Atoi(nStr)
	}

	if req.Method == http.MethodGet {
		repos, err := m.manifestHandler.List(req.Context(), n)
		if err != nil {
			return regErrInternal(err)
		}

		repositoriesToList := catalog{Repos: repos}

		msg, err := json.Marshal(repositoriesToList)
		if err != nil {
			return regErrInternal(err)
		}
		resp.Header().Set("Content-Length", strconv.Itoa(len(msg)))
		resp.WriteHeader(http.StatusOK)
		if _, err := io.Copy(resp, bytes.NewReader(msg)); err != nil {
			return regErrInternal(err)
		}
		return nil
	}

	return regErrMethodUnknown
}

// TODO: implement handling of artifactType querystring
func (m *manifests) handleReferrers(resp http.ResponseWriter, req *http.Request) *regError {
	// Ensure this is a GET request
	if req.Method != http.MethodGet {
		return regErrMethodUnknown
	}

	elem := strings.Split(req.URL.Path, "/")
	elem = elem[1:]
	target := elem[len(elem)-1]
	repo := strings.Join(elem[1:len(elem)-2], "/")

	if err := m.authz.AuthorizeReference(req.Context(), repo, target, authz.ActionRead); err != nil {
		if errors.Is(err, authz.ErrAccessDenied) {
			return regErrDenied(err.Error())
		} else if errors.Is(err, registryerror.ErrInvalidArtifactName) {
			return regErrNameInvalid
		}
		return regErrInternal(err)
	}

	// Validate that incoming target is a valid digest
	if _, err := digest.Parse(target); err != nil {
		return &regError{
			Status:  http.StatusBadRequest,
			Code:    "UNSUPPORTED",
			Message: "Target must be a valid digest",
		}
	}

	digests, err := m.manifestHandler.ListDigests(req.Context(), repo)
	if errors.Is(err, imanifest.ErrNameUnknown) {
		return regErrNameUnknown
	} else if err != nil {
		return regErrInternal(err)
	}

	im := manifest.OCI1Index{
		Index: imgspecv1.Index{
			Versioned: specs.Versioned{
				SchemaVersion: 2,
			},
			MediaType: imgspecv1.MediaTypeImageIndex,
			Manifests: []imgspecv1.Descriptor{},
		},
	}
	for _, reference := range digests {
		manifest, err := m.manifestHandler.Get(req.Context(), repo, reference.String())
		if err != nil {
			return regErrInternal(err)
		}

		var refPointer struct {
			Subject *imgspecv1.Descriptor `json:"subject"`
		}

		// TODO: remove in v2
		if len(manifest.Data) == 0 {
			b, err := m.blobHandler.Get(req.Context(), repo, manifest.Digest, false)
			if err != nil {
				return &regError{
					Status:  http.StatusNotFound,
					Code:    "BAD_REQUEST",
					Message: err.Error(),
				}
			}
			defer b.Close()
			if manifest.Data, err = io.ReadAll(b); err != nil {
				return regErrInternal(err)
			}
		}

		_ = json.Unmarshal(manifest.Data, &refPointer)
		if refPointer.Subject == nil {
			continue
		}
		referenceDigest := refPointer.Subject.Digest
		if referenceDigest.String() != target {
			continue
		}
		// At this point, we know the current digest references the target
		var imageAsArtifact struct {
			Config struct {
				MediaType string `json:"mediaType"`
			} `json:"config"`
		}
		_ = json.Unmarshal(manifest.Data, &imageAsArtifact)
		im.Manifests = append(im.Manifests, imgspecv1.Descriptor{
			MediaType:    manifest.ContentType,
			Size:         int64(len(manifest.Data)),
			Digest:       reference,
			ArtifactType: imageAsArtifact.Config.MediaType,
		})
	}
	msg, err := json.Marshal(&im)
	if err != nil {
		return regErrInternal(err)
	}
	resp.Header().Set("Content-Length", strconv.Itoa(len(msg)))
	resp.Header().Set("Content-Type", string(imgspecv1.MediaTypeImageIndex))
	resp.WriteHeader(http.StatusOK)
	if _, err := io.Copy(resp, bytes.NewReader(msg)); err != nil {
		return regErrInternal(err)
	}
	return nil
}

func (handler *manifests) handleGet(resp http.ResponseWriter, req *http.Request, repo, target string) *regError {
	ctx := req.Context()
	m, err := handler.manifestHandler.Get(ctx, repo, target)
	if errors.Is(err, imanifest.ErrNameUnknown) {
		return regErrNameUnknown
	} else if errors.Is(err, imanifest.ErrManifestUnknown) {
		return regErrManifestUnknown
	} else if err != nil {
		return regErrInternal(err)
	}

	// TODO: remove in v2
	if len(m.Data) == 0 {
		b, err := handler.blobHandler.Get(ctx, repo, m.Digest, true)
		if err != nil {
			var rerr blob.RedirectError
			if errors.As(err, &rerr) {
				if err := handler.audit.AuditPull(ctx, repo, target); err != nil {
					log := internalctx.GetLogger(ctx)
					log.Warn("failed to audit-log pull", zap.Error(err))
					sentry.GetHubFromContext(ctx).CaptureException(err)
				}
				http.Redirect(resp, req, rerr.Location, rerr.Code)
				return nil
			}
			// TODO: More nuanced
			return regErrManifestUnknown
		}
		defer b.Close()

		if m.Data, err = io.ReadAll(b); err != nil {
			return regErrInternal(err)
		}
	}

	if err := handler.audit.AuditPull(ctx, repo, target); err != nil {
		log := internalctx.GetLogger(ctx)
		log.Warn("failed to audit-log pull", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
	}

	resp.Header().Set("Docker-Content-Digest", m.Digest.String())
	resp.Header().Set("Content-Type", m.ContentType)
	resp.Header().Set("Content-Length", strconv.Itoa(len(m.Data)))
	resp.WriteHeader(http.StatusOK)
	if _, err := resp.Write(m.Data); err != nil {
		return regErrInternal(err)
	}
	return nil
}

func (handler *manifests) handleHead(resp http.ResponseWriter, req *http.Request, repo, target string) *regError {
	ctx := req.Context()
	m, err := handler.manifestHandler.Get(ctx, repo, target)
	if errors.Is(err, imanifest.ErrNameUnknown) {
		return regErrNameUnknown
	} else if errors.Is(err, imanifest.ErrManifestUnknown) {
		return regErrManifestUnknown
	} else if err != nil {
		return regErrInternal(err)
	}

	if err := handler.audit.AuditPull(ctx, repo, target); err != nil {
		log := internalctx.GetLogger(ctx)
		log.Warn("failed to audit-log pull", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
	}

	resp.Header().Set("Docker-Content-Digest", m.Digest.String())
	resp.Header().Set("Content-Type", m.ContentType)
	resp.Header().Set("Content-Length", strconv.FormatInt(m.Size, 10))
	resp.WriteHeader(http.StatusOK)
	return nil
}

func (handler *manifests) handlePut(resp http.ResponseWriter, req *http.Request, repo, target string) *regError {
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, req.Body); err != nil {
		return regErrInternal(err)
	}

	mf := imanifest.Manifest{
		ContentType: req.Header.Get("Content-Type"),
		BlobWithData: imanifest.BlobWithData{
			Data: buf.Bytes(),
			Blob: imanifest.Blob{
				Digest: digest.FromBytes(buf.Bytes()),
				Size:   int64(buf.Len()),
			},
		},
	}

	var blobs []imanifest.Blob
	if manifest.MIMETypeIsMultiImage(mf.ContentType) {
		im, err := manifest.ListFromBlob(buf.Bytes(), mf.ContentType)
		if err != nil {
			return regErrManifestInvalid(err)
		}
		for _, d := range im.Instances() {
			i, err := im.Instance(d)
			if err != nil {
				return regErrManifestInvalid(err)
			}
			blobs = append(blobs, imanifest.Blob{Digest: i.Digest, Size: i.Size})
		}
	} else {
		m, err := manifest.FromBlob(buf.Bytes(), mf.ContentType)
		if err != nil {
			return regErrManifestInvalid(err)
		}
		c := m.ConfigInfo()
		blobs = append(blobs, imanifest.Blob{Digest: c.Digest, Size: c.Size})
		for _, l := range m.LayerInfos() {
			blobs = append(blobs, imanifest.Blob{Digest: l.Digest, Size: l.Size})
		}
	}

	if err := checkIncompatibleManifest(buf.Bytes()); err != nil {
		return err
	}

	// Allow future references by target (tag) and immutable digest.
	// See https://docs.docker.com/engine/reference/commandline/pull/#pull-an-image-by-digest-immutable-identifier.
	err := db.RunTx(req.Context(), func(ctx context.Context) error {
		return multierr.Combine(
			handler.manifestHandler.Put(ctx, repo, mf.Digest.String(), mf, blobs),
			handler.manifestHandler.Put(ctx, repo, target, mf, blobs),
		)
	})
	if errors.Is(err, apierrors.ErrQuotaExceeded) {
		return regErrDeniedQuotaExceeded
	} else if errors.Is(err, imanifest.ErrTagAlreadyExists) {
		return regErrTagAlreadyExists
	} else if err != nil {
		return regErrInternal(err)
	}

	resp.Header().Set("Docker-Content-Digest", mf.Digest.String())
	resp.Header().Set("OCI-Subject", mf.Digest.String())
	resp.Header().Set("Location", req.URL.JoinPath(mf.Blob.Digest.String()).Path)
	resp.WriteHeader(http.StatusCreated)
	return nil
}

func (handler *manifests) handleDelete(resp http.ResponseWriter, req *http.Request, repo, target string) *regError {
	err := handler.manifestHandler.Delete(req.Context(), repo, target)
	if err != nil {
		if errors.Is(err, imanifest.ErrNameUnknown) {
			return regErrNameUnknown
		}
		if errors.Is(err, imanifest.ErrManifestUnknown) {
			return regErrManifestUnknown
		}
		if errors.Is(err, apierrors.ErrConflict) {
			return regErrConflict
		}
		if errors.Is(err, apierrors.ErrBadRequest) {
			return regErrBadRequest
		}
		return regErrInternal(err)
	}

	resp.WriteHeader(http.StatusAccepted)
	return nil
}

func checkIncompatibleManifest(data []byte) *regError {
	var mf struct {
		Blobs []any `json:"blobs"`
	}
	if err := json.Unmarshal(data, &mf); err != nil {
		return regErrManifestInvalid(err)
	} else if len(mf.Blobs) > 0 {
		return regErrManifestInvalid(errors.New("non-compliant manifest with blobs entry detected"))
	}
	return nil
}
