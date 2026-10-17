package handlers

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/validation"
)

// TLSAskHandler answers the Caddy on-demand TLS "ask" request (GET ...?domain=<sni>): 200 iff the
// domain is a custom domain of an organization that still exists, 404 otherwise. This is the only
// guard preventing arbitrary certificate issuance on the platform's ACME account, and it runs during
// TLS handshakes, so it must stay cheap. It is served by the internal HTTP server, which must never
// be exposed outside the cluster.
//
// Whether the domain is verified must not be part of this decision. Verification lags a corrected
// DNS record by up to a verification interval, and without a certificate the domain answers no
// https request at all in the meantime.
func TLSAskHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()

	// Domains are stored normalized, and the SNI Caddy passes here may carry a different
	// case or a trailing dot, so it must be normalized before the lookup.
	domain := validation.NormalizeHostname(r.URL.Query().Get("domain"))
	if domain == "" {
		return respond.BadRequest("parameter domain is required"), nil
	} else if exists, err := db.ExistsCustomDomain(ctx, domain); err != nil {
		return nil, fmt.Errorf("failed to check custom domain existence: %w", err)
	} else if !exists {
		return respond.Status(http.StatusNotFound), nil
	} else {
		return respond.Status(http.StatusOK), nil
	}
}
