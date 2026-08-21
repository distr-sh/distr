package turnstile

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient("test-secret", server.URL)
}

func TestVerifySendsCanonicalRequest(t *testing.T) {
	g := NewWithT(t)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.Method).To(Equal(http.MethodPost))
		g.Expect(r.Header.Get("Content-Type")).To(Equal("application/x-www-form-urlencoded"))
		g.Expect(r.ParseForm()).To(Succeed())
		g.Expect(r.PostForm.Get("secret")).To(Equal("test-secret"))
		g.Expect(r.PostForm.Get("response")).To(Equal("test-token"))
		g.Expect(r.PostForm.Get("remoteip")).To(Equal("203.0.113.1"))
		_, _ = w.Write([]byte(`{"success":true}`))
	}))

	g.Expect(client.Verify(t.Context(), "test-token", "203.0.113.1")).To(Succeed())
}

func TestVerifyRejectsUnsuccessfulResponse(t *testing.T) {
	g := NewWithT(t)

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["timeout-or-duplicate"]}`))
	}))

	g.Expect(client.Verify(t.Context(), "test-token", "")).
		To(MatchError(ContainSubstring("timeout-or-duplicate")))
}

func TestVerifyFailsClosed(t *testing.T) {
	g := NewWithT(t)

	errorResponse := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	g.Expect(errorResponse.Verify(t.Context(), "test-token", "")).
		To(MatchError(ContainSubstring("unexpected status 500")))

	malformedResponse := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	g.Expect(malformedResponse.Verify(t.Context(), "test-token", "")).
		To(MatchError(ContainSubstring("could not decode siteverify response")))

	unreachable := NewClient("test-secret", "http://127.0.0.1:1/siteverify")
	g.Expect(unreachable.Verify(t.Context(), "test-token", "")).
		To(MatchError(ContainSubstring("siteverify request failed")))
}
