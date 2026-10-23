package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/registry/authz"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	. "github.com/onsi/gomega"
)

const (
	manifestPath = "/v2/acme/app/manifests/1.0.0"
	tagsPath     = "/v2/acme/app/tags/list"
	blobPath     = "/v2/acme/app/blobs/sha256:0000000000000000000000000000000000000000000000000000000000000000"

	privatePrefix       = "/v2/acme/private/"
	privateManifestPath = privatePrefix + "manifests/1.0.0"
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
	// Stands in for the authorizer, which refuses an anonymous request for a private artifact and
	// spends the budget only on the ones it grants.
	authorize := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.authenticated {
			if strings.HasPrefix(r.URL.Path, privatePrefix) {
				_ = regErrAuthz(authz.ErrAuthenticationRequired).Write(w, r)
				return
			} else if authz.AnonymousLimitExceeded(r.Context()) {
				_ = regErrAuthz(authz.ErrRateLimited).Write(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})
	h.handler = passwordGrant(
		middleware.Split(requiresAuthentication, marker, rateLimitAnonymous(limits))(authorize),
	)
	return h
}

func (h *registryHandler) serve(path string, basicAuth ...string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	if len(basicAuth) == 2 {
		r.SetBasicAuth(basicAuth[0], basicAuth[1])
	}
	return h.serveRequest(r)
}

func (h *registryHandler) serveRequest(r *http.Request) *httptest.ResponseRecorder {
	h.authenticated = false
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
		authorization string
		form          url.Values
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
		{
			name:          "manifest with the anonymous token",
			method:        http.MethodGet,
			path:          manifestPath,
			authorization: "Bearer " + auth.AnonymousRegistryToken,
			authenticated: false,
		},
		{
			name:          "manifest with a token",
			method:        http.MethodGet,
			path:          manifestPath,
			authorization: "Bearer pat",
			authenticated: true,
		},
		{
			name:          "ping with the anonymous token keeps its challenge",
			method:        http.MethodGet,
			path:          "/v2/",
			authorization: "Bearer " + auth.AnonymousRegistryToken,
			authenticated: true,
		},
		{
			// The legacy Docker pull fetches a token before every pull, so without credentials it
			// has to get the anonymous one rather than a 401.
			name:          "token without credentials",
			method:        http.MethodGet,
			path:          tokenPath,
			authenticated: false,
		},
		{
			name:          "token with credentials",
			method:        http.MethodGet,
			path:          tokenPath,
			basicAuth:     []string{"user", "pat"},
			authenticated: true,
		},
		{
			// containerd sends its credentials as an OAuth2 password grant first.
			name:          "token with a password grant",
			method:        http.MethodPost,
			path:          tokenPath,
			form:          url.Values{"grant_type": {"password"}, "username": {"user"}, "password": {"pat"}},
			authenticated: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var r *http.Request
			if tt.form != nil {
				r = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.form.Encode()))
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				r = httptest.NewRequest(tt.method, tt.path, nil)
			}
			if len(tt.basicAuth) == 2 {
				r.SetBasicAuth(tt.basicAuth[0], tt.basicAuth[1])
			} else if tt.authorization != "" {
				r.Header.Set("Authorization", tt.authorization)
			}
			handler.serveRequest(r)
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
	// A Basic challenge next to the Bearer one would keep the legacy Docker pull from pulling a public
	// artifact anonymously, and oras-go only sends credentials to an absolute realm on the same host.
	g.Expect(w.Header().Values("WWW-Authenticate")).
		To(Equal([]string{`Bearer realm="https://example.com/v2/token",service="example.com"`}))

	var body struct {
		Errors []struct {
			Code string `json:"code"`
		} `json:"errors"`
	}
	g.Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
	g.Expect(body.Errors).To(HaveLen(1))
	g.Expect(body.Errors[0].Code).To(Equal("UNAUTHORIZED"))
}

func TestTokenResponse(t *testing.T) {
	t.Run("anonymous token", func(t *testing.T) {
		g := NewWithT(t)
		w := httptest.NewRecorder()
		g.Expect(handleToken(w, httptest.NewRequest(http.MethodGet, tokenPath, nil))).To(BeNil())

		var body map[string]any
		g.Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		// containerd and oras-go read only access_token from an OAuth2 response, the legacy Docker
		// pull reads either, and docker login would store a refresh token in place of the PAT.
		g.Expect(body).To(HaveKeyWithValue("token", auth.AnonymousRegistryToken))
		g.Expect(body).To(HaveKeyWithValue("access_token", auth.AnonymousRegistryToken))
		g.Expect(body).To(HaveKeyWithValue("expires_in", BeNumerically(">", 0)))
		g.Expect(body).NotTo(HaveKey("refresh_token"))
	})

	t.Run("unsupported grant type", func(t *testing.T) {
		g := NewWithT(t)
		handler := newRegistryHandler(env.AnonymousRateLimits{})
		r := httptest.NewRequest(http.MethodPost, tokenPath,
			strings.NewReader(url.Values{"grant_type": {"refresh_token"}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// containerd falls back to a GET with Basic auth on a 400.
		g.Expect(handler.serveRequest(r).Code).To(Equal(http.StatusBadRequest))
	})
}

func TestAnonymousAccessRateLimit(t *testing.T) {
	t.Run("manifests and blobs have separate budgets", func(t *testing.T) {
		g := NewWithT(t)
		handler := newRegistryHandler(env.AnonymousRateLimits{ManifestsPerHour: 2, BlobsPerHour: 1})

		g.Expect(handler.serve(manifestPath).Code).To(Equal(http.StatusOK))
		g.Expect(handler.serve(tagsPath).Code).
			To(Equal(http.StatusOK), "a listing shares the manifest budget")
		g.Expect(handler.serve(blobPath).Code).
			To(Equal(http.StatusOK), "the blob budget is untouched by the manifest requests")

		w := handler.serve(manifestPath)
		// An OCI client sends its credentials only after a 401 and returns every other status to its
		// caller, so a 429 would lock a client that holds credentials out of a public artifact.
		g.Expect(w.Code).To(Equal(http.StatusUnauthorized))
		g.Expect(w.Header().Get("WWW-Authenticate")).NotTo(BeEmpty())
		g.Expect(w.Header().Get("Retry-After")).NotTo(BeEmpty())

		var body struct {
			Errors []struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"errors"`
		}
		g.Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		g.Expect(body.Errors).To(HaveLen(1))
		g.Expect(body.Errors[0].Message).To(ContainSubstring("rate limit"),
			"a client without credentials has only the message to tell it apart from a private artifact")

		g.Expect(handler.serve(blobPath).Code).To(Equal(http.StatusUnauthorized))
	})

	t.Run("authenticated requests are not counted", func(t *testing.T) {
		g := NewWithT(t)
		handler := newRegistryHandler(env.AnonymousRateLimits{ManifestsPerHour: 1})

		for range 3 {
			g.Expect(handler.serve(manifestPath, "user", "pat").Code).To(Equal(http.StatusOK))
		}
	})

	t.Run("refused requests are not counted", func(t *testing.T) {
		g := NewWithT(t)
		handler := newRegistryHandler(env.AnonymousRateLimits{ManifestsPerHour: 1})

		// Helm and other OCI clients send every pull of a private artifact without credentials
		// first and authenticate only after the challenge.
		for range 3 {
			w := handler.serve(privateManifestPath)
			g.Expect(w.Code).To(Equal(http.StatusUnauthorized))
			g.Expect(w.Header().Get("WWW-Authenticate")).NotTo(BeEmpty())
		}
		g.Expect(handler.serve(manifestPath).Code).
			To(Equal(http.StatusOK), "the refused pulls left the budget untouched")
	})

	t.Run("a zero limit is disabled", func(t *testing.T) {
		g := NewWithT(t)
		handler := newRegistryHandler(env.AnonymousRateLimits{})

		for range 3 {
			g.Expect(handler.serve(manifestPath).Code).To(Equal(http.StatusOK))
		}
	})
}
