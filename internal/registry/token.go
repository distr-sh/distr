package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/handlerutil"
)

const tokenPath = "/v2/token"

const tokenLifetimeSeconds = 60

const maxTokenFormBytes = 8 << 10

type tokenResponse struct {
	Token       string    `json:"token"`
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	IssuedAt    time.Time `json:"issued_at"`
}

func isToken(r *http.Request) bool {
	return r.URL.Path == tokenPath
}

// bearerChallenge points the client at the token endpoint on the host it used for this request.
// oras-go refuses to send credentials to a realm on a private address unless the realm has the
// hostname of the registry, so the realm must not name another host.
//
// The challenge has to be Bearer alone. The legacy Docker pull runs the handler of every challenge
// it was given, and its Basic handler fails without credentials, which is what kept it from pulling
// a public artifact anonymously.
func bearerChallenge(r *http.Request) string {
	return fmt.Sprintf(`Bearer realm="%v%v",service="%v"`, handlerutil.GetRequestSchemeAndHost(r), tokenPath, r.Host)
}

// passwordGrant moves the credentials of an OAuth2 password grant, which containerd sends before it
// falls back to a GET with Basic auth, into the Basic auth header the authentication middleware reads.
func passwordGrant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isToken(r) || r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxTokenFormBytes)
		if err := r.ParseForm(); err != nil {
			_ = regErrBadRequest.Write(w, r)
		} else if r.PostForm.Get("grant_type") != "password" {
			_ = regErrUnsupportedGrantType.Write(w, r)
		} else {
			r = r.Clone(r.Context())
			r.SetBasicAuth(r.PostForm.Get("username"), r.PostForm.Get("password"))
			next.ServeHTTP(w, r)
		}
	})
}

// handleToken answers the token request of the Bearer challenge. The token of an authenticated
// request is the credential it authenticated with, so that every registry request is still checked
// against the database and a revoked PAT is refused at once. No refresh token is handed out, since
// docker login would store it instead of the PAT.
func handleToken(resp http.ResponseWriter, req *http.Request) *regError {
	if req.Method != http.MethodGet && req.Method != http.MethodPost {
		return regErrMethodUnknown
	}
	token := auth.AnonymousRegistryToken
	if _, ok := auth.ArtifactsPrincipal(req.Context()); ok {
		token = auth.ArtifactsCredential(req)
	}
	if body, err := json.Marshal(tokenResponse{
		Token:       token,
		AccessToken: token,
		ExpiresIn:   tokenLifetimeSeconds,
		IssuedAt:    time.Now().UTC(),
	}); err != nil {
		return regErrInternal(err)
	} else {
		resp.Header().Set("Content-Type", "application/json")
		resp.Header().Set("Cache-Control", "no-store")
		_, _ = resp.Write(body)
		return nil
	}
}
