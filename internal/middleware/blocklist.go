package middleware

import (
	"net/http"

	"github.com/distr-sh/distr/internal/blocklist"
)

func BlockIPs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if blocklist.ClientIPBlocked(r.Context()) {
			http.Error(w, blocklist.IPBlockedMessage, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
