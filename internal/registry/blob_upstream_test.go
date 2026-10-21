package registry

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/registry/blob"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	"go.uber.org/zap"
)

type fakeUpstream struct{ content io.Reader }

func (f *fakeUpstream) OpenBlob(context.Context, string, digest.Digest) (int64, io.ReadCloser, error) {
	return 0, io.NopCloser(f.content), nil
}

type memoryBlobStore struct {
	blob.BlobPutHandler
	lock  sync.Mutex
	blobs map[digest.Digest][]byte
}

func (s *memoryBlobStore) Get(context.Context, string, digest.Digest, bool) (io.ReadCloser, error) {
	return nil, blob.ErrNotFound
}

func (s *memoryBlobStore) Put(_ context.Context, _ string, h digest.Digest, _ string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.blobs[h] = data
	return nil
}

func (s *memoryBlobStore) stored(h digest.Digest) []byte {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.blobs[h]
}

type disconnectedClient struct{ header http.Header }

func (c *disconnectedClient) Header() http.Header       { return c.header }
func (c *disconnectedClient) WriteHeader(int)           {}
func (c *disconnectedClient) Write([]byte) (int, error) { return 0, errors.New("client disconnected") }

func serveUpstreamBlob(resp http.ResponseWriter, content io.Reader, h digest.Digest) *memoryBlobStore {
	store := &memoryBlobStore{blobs: map[digest.Digest][]byte{}}
	b := &blobs{blobHandler: store, upstreamFetcher: &fakeUpstream{content: content}}
	req := httptest.NewRequestWithContext(
		internalctx.WithLogger(context.Background(), zap.NewNop()), http.MethodGet, "/", nil)
	Expect(b.serveFromUpstream(resp, req, "acme/app", h)).To(BeNil())
	return store
}

func TestServeFromUpstreamStoresBlobAfterClientDisconnected(t *testing.T) {
	RegisterTestingT(t)
	content := bytes.Repeat([]byte("layer"), 100_000)
	h := digest.FromBytes(content)

	store := serveUpstreamBlob(&disconnectedClient{header: http.Header{}}, bytes.NewReader(content), h)

	Eventually(func() []byte { return store.stored(h) }).WithTimeout(5 * time.Second).Should(Equal(content))
}

func TestServeFromUpstreamDoesNotStoreFailedDownload(t *testing.T) {
	RegisterTestingT(t)
	content := bytes.Repeat([]byte("layer"), 100_000)
	h := digest.FromBytes(content)
	failing := io.MultiReader(bytes.NewReader(content[:1000]), iotest.ErrReader(io.ErrUnexpectedEOF))
	resp := httptest.NewRecorder()

	store := serveUpstreamBlob(resp, failing, h)

	Expect(resp.Body.Bytes()).To(Equal(content[:1000]))
	Consistently(func() []byte { return store.stored(h) }).WithTimeout(200 * time.Millisecond).Should(BeNil())
}
