package middleware

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/buildconfig"
)

func CacheControl(maxAge time.Duration, filterFunc func(r *http.Request) bool) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if filterFunc == nil || filterFunc(r) {
				w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%v", maxAge.Seconds()))
			}
			next.ServeHTTP(w, r)
		})
	}
}

var (
	bundleAssetExtensions = []string{".css", ".js"}
	mediaAssetExtensions  = []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".woff2"}
)

// CacheControlBundleAssets sets Cache-Control header for bundle assets (CSS and JS) to 24 hours in release builds.
// Cache-Control is disabled in dev builds because the main and styles bundles are not hashed
var CacheControlBundleAssets = CacheControl(24*time.Hour, func(r *http.Request) bool {
	return buildconfig.IsRelease() && hasExtension(r.URL.Path, bundleAssetExtensions)
})

// CacheControlMediaAssets sets Cache-Control header for media assets (images and fonts) to 1 hour in release builds.
// Cache-Control is disabled in dev builds because media assets might change
var CacheControlMediaAssets = CacheControl(time.Hour, func(r *http.Request) bool {
	return buildconfig.IsRelease() && hasExtension(r.URL.Path, mediaAssetExtensions)
})

func IsAssetPath(path string) bool {
	return hasExtension(path, bundleAssetExtensions) || hasExtension(path, mediaAssetExtensions)
}

func hasExtension(path string, extensions []string) bool {
	return slices.ContainsFunc(extensions, func(ext string) bool { return strings.HasSuffix(path, ext) })
}
