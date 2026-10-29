package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

type countingReader struct {
	io.Reader
	read int
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.read += n
	return n, err
}

func TestRequestSize(t *testing.T) {
	const payload = "longer than the limit"
	var readErr error
	handler := RequestSize(16)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))

	t.Run("declared length is rejected without reading the body", func(t *testing.T) {
		g := NewWithT(t)
		body := &countingReader{Reader: strings.NewReader(payload)}
		req := httptest.NewRequest(http.MethodPut, "/", body)
		req.ContentLength = int64(len(payload))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		g.Expect(w.Code).To(Equal(http.StatusRequestEntityTooLarge))
		g.Expect(body.read).To(BeZero())
	})

	t.Run("undeclared length is cut off while reading", func(t *testing.T) {
		g := NewWithT(t)
		req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(payload))
		req.ContentLength = -1
		handler.ServeHTTP(httptest.NewRecorder(), req)
		_, ok := errors.AsType[*http.MaxBytesError](readErr)
		g.Expect(ok).To(BeTrue())
	})
}
