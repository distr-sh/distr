package registry

import (
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/registry/authz"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func requiresAuthentication(r *http.Request) bool {
	return hasCredentials(r) || !allowsAnonymous(r)
}

// useRegistryErrorFormat replaces the plain text 401 of the authentication middleware with the error
// format of the distribution spec, which is what an OCI client reads.
func useRegistryErrorFormat() {
	auth.ArtifactsAuthentication.SetUnauthorizedHandler(func(w http.ResponseWriter, r *http.Request) {
		_ = regErrCredentialsRequired.Write(w)
	})
}

type anonymousLimiter struct {
	pred    func(*http.Request) bool
	limiter *httprate.RateLimiter
}

// rateLimitAnonymous attaches the anonymous pull budget of the client IP to the request, which the
// authorizer spends only on requests it grants (see authz.WithAnonymousLimit).
func rateLimitAnonymous(limits env.AnonymousRateLimits) func(http.Handler) http.Handler {
	var limiters []anonymousLimiter
	for _, l := range []struct {
		pred   func(*http.Request) bool
		limit  int
		window time.Duration
	}{
		{isManifestOrListing, limits.ManifestsPerMinute, time.Minute},
		{isManifestOrListing, limits.ManifestsPerHour, time.Hour},
		{isBlob, limits.BlobsPerMinute, time.Minute},
		{isBlob, limits.BlobsPerHour, time.Hour},
	} {
		if l.limit > 0 {
			limiters = append(limiters, anonymousLimiter{l.pred, httprate.NewRateLimiter(l.limit, l.window)})
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := authz.WithAnonymousLimit(r.Context(), func() bool {
				key := clientIPKey(r)
				for _, l := range limiters {
					if l.pred(r) && l.limiter.OnLimit(w, r, key) {
						return true
					}
				}
				return false
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// hasCredentials reports whether the request is worth authenticating. An OCI client that has no
// credentials for the registry still sends an empty Basic header when it sees a Basic challenge
// (containers/image does), so that counts as anonymous rather than as a failed login.
func hasCredentials(r *http.Request) bool {
	if _, password, ok := r.BasicAuth(); ok {
		return password != ""
	}
	return r.Header.Get("Authorization") != ""
}

// allowsAnonymous lists the routes that name an artifact, which is what makes the authorizer able
// to decide anonymous access at all. The catalog belongs to the organization rather than to an
// artifact and must stay out, as must the ping: OCI clients record the authentication challenge
// from the /v2/ response alone, so a 200 there would leave a client that holds credentials with no
// challenge to answer and it would never send them, breaking every private pull.
func allowsAnonymous(r *http.Request) bool {
	return isManifest(r) || isBlob(r) || isTags(r) || isReferrers(r)
}

// isManifestOrListing groups the requests a client makes to find out what an artifact holds, which
// are the ones served from the database, as opposed to the blob requests served from S3.
func isManifestOrListing(r *http.Request) bool {
	return isManifest(r) || isTags(r) || isReferrers(r)
}

// clientIPKey keys the limiter by the IP that chi's ClientIPFrom* middlewares resolved.
// CanonicalizeIP reduces an IPv6 client to its /64, without which it would rotate addresses inside
// its own prefix to win a fresh bucket per request.
func clientIPKey(r *http.Request) string {
	return httprate.CanonicalizeIP(chimiddleware.GetClientIP(r.Context()))
}
