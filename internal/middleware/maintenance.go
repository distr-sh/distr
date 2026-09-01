package middleware

import (
	"net/http"

	"github.com/distr-sh/distr/internal/env"
)

const maintenanceMessage = "Distr is temporarily unavailable while maintenance is being performed"

// MaintenanceMode answers every request with 503 while the instance is in maintenance mode, so that
// nothing but the frontend reaches the database during a maintenance window.
func MaintenanceMode(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if env.MaintenanceMode() {
			http.Error(w, maintenanceMessage, http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, r)
	})
}
