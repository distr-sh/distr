package middleware

import (
	"net/http"

	"github.com/distr-sh/distr/internal/blocklist"
	"github.com/distr-sh/distr/internal/respond"
)

var BlockIPs = guard(func(r *http.Request) (respond.Responder, error) {
	if blocklist.ClientIPBlocked(r.Context()) {
		return respond.Forbidden(blocklist.IPBlockedMessage), nil
	}
	return nil, nil
})
