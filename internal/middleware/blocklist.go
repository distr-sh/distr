package middleware

import (
	"net/http"

	"github.com/distr-sh/distr/internal/blocklist"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func BlockIPs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if blocklist.IPBlocked(chimiddleware.GetClientIPAddr(r.Context())) {
			http.Error(w, blocklist.IPBlockedMessage, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
