package handlers

import (
	"net/http"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// TLSAskHandler answers the Caddy on-demand TLS "ask" request (GET ...?domain=<sni>): 200 iff
// the domain is a registered custom domain, 404 otherwise. This is the only guard preventing
// arbitrary certificate issuance on the platform's ACME account, and it runs during TLS
// handshakes, so it must stay a single indexed lookup. It is served by the internal HTTP
// server, which must never be exposed outside the cluster.
func TLSAskHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		domain := normalizeHost(r.FormValue("domain"))
		if domain == "" {
			http.Error(w, "parameter domain is required", http.StatusBadRequest)
			return
		}

		if exists, err := db.ExistsCustomDomain(ctx, domain); err != nil {
			internalctx.GetLogger(ctx).Error("failed to check custom domain existence", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else if !exists {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}
