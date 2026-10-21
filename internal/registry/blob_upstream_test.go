package registry

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/iotest"
	"time"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/registry/blob"
	"github.com/distr-sh/distr/internal/registry/blob/inmemory"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	"go.uber.org/zap"
)

type fakeUpstream struct{ content io.Reader }

func (f *fakeUpstream) OpenBlob(context.Context, string, digest.Digest) (int64, io.ReadCloser, error) {
	return 0, io.NopCloser(f.content), nil
}

func readBlob(store blob.BlobHandler, h digest.Digest) ([]byte, error) {
	rc, err := store.Get(context.Background(), "acme/app", h, true)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

type disconnectedClient struct{ header http.Header }

func (c *disconnectedClient) Header() http.Header       { return c.header }
func (c *disconnectedClient) WriteHeader(int)           {}
func (c *disconnectedClient) Write([]byte) (int, error) { return 0, errors.New("client disconnected") }

// stalledClient keeps the connection open without reading, so every write blocks until the write deadline passes.
type stalledClient struct {
	header   http.Header
	deadline time.Time
}

func (c *stalledClient) Header() http.Header { return c.header }
func (c *stalledClient) WriteHeader(int)     {}

func (c *stalledClient) Write([]byte) (int, error) {
	if c.deadline.IsZero() {
		select {}
	}
	time.Sleep(time.Until(c.deadline))
	return 0, os.ErrDeadlineExceeded
}

func (c *stalledClient) SetWriteDeadline(deadline time.Time) error {
	c.deadline = deadline
	return nil
}

func serveUpstreamBlob(resp http.ResponseWriter, content io.Reader, h digest.Digest) blob.BlobHandler {
	store := inmemory.NewBlobHandler()
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

	Eventually(func() ([]byte, error) { return readBlob(store, h) }).
		WithTimeout(5 * time.Second).Should(Equal(content))
}

func TestServeFromUpstreamStoresBlobForStalledClient(t *testing.T) {
	RegisterTestingT(t)
	timeout := upstreamBlobClientWriteTimeout
	upstreamBlobClientWriteTimeout = 50 * time.Millisecond
	t.Cleanup(func() { upstreamBlobClientWriteTimeout = timeout })
	content := bytes.Repeat([]byte("layer"), 100_000)
	h := digest.FromBytes(content)
	client := &stalledClient{header: http.Header{}}

	store := serveUpstreamBlob(client, bytes.NewReader(content), h)

	Expect(client.deadline.IsZero()).To(BeTrue())
	Eventually(func() ([]byte, error) { return readBlob(store, h) }).
		WithTimeout(5 * time.Second).Should(Equal(content))
}

func TestServeFromUpstreamDoesNotStoreFailedDownload(t *testing.T) {
	RegisterTestingT(t)
	content := bytes.Repeat([]byte("layer"), 100_000)
	h := digest.FromBytes(content)
	failing := io.MultiReader(bytes.NewReader(content[:1000]), iotest.ErrReader(io.ErrUnexpectedEOF))
	resp := httptest.NewRecorder()

	store := serveUpstreamBlob(resp, failing, h)

	Expect(resp.Body.Bytes()).To(Equal(content[:1000]))
	Consistently(func() error {
		_, err := readBlob(store, h)
		return err
	}).WithTimeout(200 * time.Millisecond).Should(MatchError(blob.ErrNotFound))
}
