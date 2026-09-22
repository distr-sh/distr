package registry

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/middleware"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	. "github.com/onsi/gomega"
)

const (
	manifestPath = "/v2/acme/app/manifests/1.0.0"
	tagsPath     = "/v2/acme/app/tags/list"
	blobPath     = "/v2/acme/app/blobs/sha256:0000000000000000000000000000000000000000000000000000000000000000"
)

// registryHandler composes the registry's middleware split with a marker in place of the real
// authentication chain, which records whether a request was sent down it instead of being served
// anonymously.
type registryHandler struct {
	handler       http.Handler
	authenticated bool
}

func newRegistryHandler(limits env.AnonymousRateLimits) *registryHandler {
	h := &registryHandler{}
	marker := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.authenticated = true
			next.ServeHTTP(w, r)
		})
	}
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h.handler = middleware.Split(requiresAuthentication, marker, rateLimitAnonymous(limits))(ok)
	return h
}

func (h *registryHandler) serve(method, path string, basicAuth ...string) *httptest.ResponseRecorder {
	h.authenticated = false
	r := httptest.NewRequest(method, path, nil)
	if len(basicAuth) == 2 {
		r.Header.Set("Authorization", "Basic "+
			base64.StdEncoding.EncodeToString([]byte(basicAuth[0]+":"+basicAuth[1])))
	}
	w := httptest.NewRecorder()
	// The limiter keys on the client IP the chi middlewares resolve, which a bare request lacks.
	chimiddleware.ClientIPFromRemoteAddr(h.handler).ServeHTTP(w, r)
	return w
}

func TestAnonymousAccessRouting(t *testing.T) {
	g := NewWithT(t)
	handler := newRegistryHandler(env.AnonymousRateLimits{})

	for _, tt := range []struct {
		name          string
		method        string
		path          string
		basicAuth     []string
		authenticated bool
	}{
		{name: "manifest without credentials", method: http.MethodGet, path: manifestPath},
		{name: "blob without credentials", method: http.MethodGet, path: blobPath},
		{name: "tag list without credentials", method: http.MethodGet, path: tagsPath},
		{
			name:   "referrers without credentials",
			method: http.MethodGet,
			path:   "/v2/acme/app/referrers/sha256:0000",
		},
		{
			name:   "manifest push without credentials reaches the authorizer, which refuses it",
			method: http.MethodPut,
			path:   manifestPath,
		},
		{
			// A 200 on the ping would leave a client with no challenge to answer, so it would never
			// send the credentials it has and every private pull would fail.
			name:          "ping keeps its challenge",
			method:        http.MethodGet,
			path:          "/v2/",
			authenticated: true,
		},
		{
			name:          "catalog is never anonymous",
			method:        http.MethodGet,
			path:          "/v2/_catalog",
			authenticated: true,
		},
		{
			name:          "manifest with credentials",
			method:        http.MethodGet,
			path:          manifestPath,
			basicAuth:     []string{"user", "pat"},
			authenticated: true,
		},
		{
			// containers/image sends an empty Basic header when it sees a Basic challenge and holds
			// no credentials, which is an anonymous request rather than a failed login.
			name:          "manifest with empty basic credentials",
			method:        http.MethodGet,
			path:          manifestPath,
			basicAuth:     []string{"", ""},
			authenticated: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handler.serve(tt.method, tt.path, tt.basicAuth...)
			g.Expect(handler.authenticated).To(Equal(tt.authenticated))
		})
	}
}

func TestUnauthenticatedResponse(t *testing.T) {
	g := NewWithT(t)
	useRegistryErrorFormat()
	served := false
	handler := auth.ArtifactsAuthentication.Middleware(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { served = true }),
	)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v2/", nil))

	g.Expect(served).To(BeFalse())
	g.Expect(w.Code).To(Equal(http.StatusUnauthorized))
	g.Expect(w.Header().Get("WWW-Authenticate")).
		NotTo(BeEmpty(), "a client sends its credentials only after it has seen the challenge")

	var body struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	g.Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
	g.Expect(body.Errors).To(HaveLen(1))
	g.Expect(body.Errors[0].Code).To(Equal("UNAUTHORIZED"))
	g.Expect(body.Errors[0].Message).To(ContainSubstring("containerd image store"))
}

func TestAnonymousAccessRateLimit(t *testing.T) {
	t.Run("manifests and blobs have separate budgets", func(t *testing.T) {
		g := NewWithT(t)
		handler := newRegistryHandler(env.AnonymousRateLimits{ManifestsPerHour: 2, BlobsPerHour: 1})

		g.Expect(handler.serve(http.MethodGet, manifestPath).Code).To(Equal(http.StatusOK))
		g.Expect(handler.serve(http.MethodGet, tagsPath).Code).
			To(Equal(http.StatusOK), "a listing shares the manifest budget")
		g.Expect(handler.serve(http.MethodGet, blobPath).Code).
			To(Equal(http.StatusOK), "the blob budget is untouched by the manifest requests")

		w := handler.serve(http.MethodGet, manifestPath)
		g.Expect(w.Code).To(Equal(http.StatusTooManyRequests))
		g.Expect(w.Header().Get("Retry-After")).NotTo(BeEmpty())

		var body struct {
			Errors []struct {
				Code string `json:"code"`
			} `json:"errors"`
		}
		g.Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		g.Expect(body.Errors).To(HaveLen(1))
		g.Expect(body.Errors[0].Code).To(Equal("TOOMANYREQUESTS"), "clients back off on the OCI error code")

		g.Expect(handler.serve(http.MethodGet, blobPath).Code).To(Equal(http.StatusTooManyRequests))
	})

	t.Run("authenticated requests are not counted", func(t *testing.T) {
		g := NewWithT(t)
		handler := newRegistryHandler(env.AnonymousRateLimits{ManifestsPerHour: 1})

		for range 3 {
			g.Expect(handler.serve(http.MethodGet, manifestPath, "user", "pat").Code).To(Equal(http.StatusOK))
		}
	})

	t.Run("a zero limit is disabled", func(t *testing.T) {
		g := NewWithT(t)
		handler := newRegistryHandler(env.AnonymousRateLimits{})

		for range 3 {
			g.Expect(handler.serve(http.MethodGet, manifestPath).Code).To(Equal(http.StatusOK))
		}
	})
}
