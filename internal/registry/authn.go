package registry

import (
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/middleware"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

func requiresAuthentication(r *http.Request) bool {
	return hasCredentials(r) || !allowsAnonymous(r)
}

func rateLimitAnonymous(limits env.AnonymousRateLimits) func(http.Handler) http.Handler {
	return middleware.Chain(
		limitAnonymous(isManifestOrListing, limits.ManifestsPerMinute, time.Minute),
		limitAnonymous(isManifestOrListing, limits.ManifestsPerHour, time.Hour),
		limitAnonymous(isBlob, limits.BlobsPerMinute, time.Minute),
		limitAnonymous(isBlob, limits.BlobsPerHour, time.Hour),
	)
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

// limitAnonymous rate limits the requests matching pred by client IP. A limit of zero is disabled.
func limitAnonymous(
	pred func(*http.Request) bool,
	limit int,
	window time.Duration,
) func(http.Handler) http.Handler {
	if limit <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	limiter := httprate.LimitBy(limit, window, clientIPKey, httprate.WithLimitHandler(writeTooManyRequests))
	return func(next http.Handler) http.Handler {
		limited := limiter(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if pred(r) {
				limited.ServeHTTP(w, r)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// clientIPKey keys the limiter by the IP that chi's ClientIPFrom* middlewares resolved.
// CanonicalizeIP reduces an IPv6 client to its /64, without which it would rotate addresses inside
// its own prefix to win a fresh bucket per request.
func clientIPKey(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(chimiddleware.GetClientIP(r.Context())), nil
}

func writeTooManyRequests(w http.ResponseWriter, r *http.Request) {
	_ = regErrTooManyRequests.Write(w)
}
