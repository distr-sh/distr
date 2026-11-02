package inmemory

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/distr-sh/distr/internal/registry/and"
	"github.com/distr-sh/distr/internal/registry/blob"
	"github.com/distr-sh/distr/internal/registry/verify"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
)

type blobHandler struct {
	m        map[string][]byte
	sessions map[string][]byte
	lock     sync.Mutex
}

func NewBlobHandler() blob.BlobHandler {
	return &blobHandler{m: map[string][]byte{}, sessions: map[string][]byte{}}
}

func (m *blobHandler) Stat(_ context.Context, _ string, h digest.Digest) (int64, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	b, found := m.m[h.String()]
	if !found {
		return 0, blob.ErrNotFound
	}
	return int64(len(b)), nil
}

func (m *blobHandler) Get(_ context.Context, _ string, h digest.Digest, _ bool) (io.ReadCloser, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	b, found := m.m[h.String()]
	if !found {
		return nil, blob.ErrNotFound
	}
	return &and.BytesCloser{Reader: bytes.NewReader(b)}, nil
}

func (m *blobHandler) Put(_ context.Context, _ string, h digest.Digest, _ string, r io.Reader) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if rc, ok := r.(io.ReadCloser); ok {
		defer rc.Close()
	}
	all, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.m[h.String()] = all
	return nil
}

func (m *blobHandler) StartSession(context.Context, string) (string, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	id := uuid.NewString()
	m.sessions[id] = nil
	return id, nil
}

func (m *blobHandler) PutChunk(_ context.Context, id string, r io.Reader, start int64) (int64, error) {
	if rc, ok := r.(io.Closer); ok {
		defer rc.Close()
	}
	chunk, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	uploaded, found := m.sessions[id]
	if !found {
		return 0, blob.NewErrBadUpload("unknown upload")
	} else if start != int64(len(uploaded)) {
		return 0, blob.NewErrBadUpload("range is not as expected")
	}
	m.sessions[id] = append(uploaded, chunk...)
	return int64(len(m.sessions[id])), nil
}

func (m *blobHandler) GetUploadedPartsSize(_ context.Context, id string) (int64, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if uploaded, found := m.sessions[id]; !found {
		return 0, blob.NewErrBadUpload("unknown upload")
	} else {
		return int64(len(uploaded)), nil
	}
}

func (m *blobHandler) CompleteSession(_ context.Context, _, id string, h digest.Digest) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	uploaded, found := m.sessions[id]
	if !found {
		return blob.NewErrBadUpload("unknown upload")
	} else if len(uploaded) == 0 {
		return blob.NewErrBadUpload("upload has no chunks")
	}

	delete(m.sessions, id)
	verified, err := verify.ReadCloser(io.NopCloser(bytes.NewReader(uploaded)), int64(len(uploaded)), h)
	if err != nil {
		return err
	}
	defer verified.Close()
	if _, err := io.Copy(io.Discard, verified); err != nil {
		return err
	}
	m.m[h.String()] = uploaded
	return nil
}

func (m *blobHandler) Delete(_ context.Context, _ string, h digest.Digest) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, found := m.m[h.String()]; !found {
		return blob.ErrNotFound
	}

	delete(m.m, h.String())
	return nil
}
