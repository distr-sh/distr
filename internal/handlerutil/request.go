package handlerutil

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/env"
)

// GetRequestSchemeAndHost is the base URL of the current request: the host it was sent to, so that a
// request on a custom domain keeps that domain, with the scheme this instance is configured with, since
// a request forwarded by a TLS-terminating proxy arrives as plain http.
func GetRequestSchemeAndHost(r *http.Request) string {
	return fmt.Sprintf("%v://%v", env.HostScheme(), r.Host)
}
