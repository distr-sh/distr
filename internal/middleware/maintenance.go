package middleware

import (
	"net/http"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/respond"
)

var maintenanceInProgress = respond.Error(
	http.StatusServiceUnavailable,
	"Distr is temporarily unavailable while maintenance is being performed",
)

// MaintenanceMode answers every request with 503 while the instance is in maintenance mode, so that
// nothing but the frontend reaches the database during a maintenance window.
var MaintenanceMode = guard(func(r *http.Request) (respond.Responder, error) {
	if env.MaintenanceMode() {
		return maintenanceInProgress, nil
	}
	return nil, nil
})
