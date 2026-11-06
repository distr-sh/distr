package routing

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	. "github.com/onsi/gomega"
)

func TestStaticFileHandler(t *testing.T) {
	handler := staticFileHandler(fstest.MapFS{
		"index.html": {Data: []byte("<app-root></app-root>")},
		"chunk-a.js": {Data: []byte("export {};")},
	})
	serve := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		return w
	}

	t.Run("missing asset is not answered with index.html", func(t *testing.T) {
		g := NewWithT(t)
		w := serve("/chunk-missing.js")
		g.Expect(w.Code).To(Equal(http.StatusNotFound))
		g.Expect(w.Header().Get("Cache-Control")).To(Equal("no-store"))
	})

	for _, path := range []string{"/", "/deployments"} {
		t.Run("route "+path+" serves uncached index.html", func(t *testing.T) {
			g := NewWithT(t)
			w := serve(path)
			g.Expect(w.Code).To(Equal(http.StatusOK))
			g.Expect(w.Body.String()).To(Equal("<app-root></app-root>"))
			g.Expect(w.Header().Get("Cache-Control")).To(Equal("no-cache"))
		})
	}
}
