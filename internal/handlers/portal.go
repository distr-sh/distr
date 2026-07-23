package handlers

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/distr-sh/distr/api"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func PublicPortalRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Portal"))
	r.Get("/", getPortalHandler).
		With(option.Description("Get host-resolved portal branding (browser tab title, favicon and logo)")).
		With(option.Response(http.StatusOK, api.PortalResponse{})).
		With(option.Response(http.StatusNoContent, nil,
			option.ContentDescription("The host did not resolve to a custom app domain, or branding could not be "+
				"resolved. Clients are instructed to apply default branding.")))
}

func getPortalHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	host := normalizeHost(r.Host)

	var response *api.PortalResponse
	if branding, err := resolvePortalBranding(ctx, host); err != nil {
		// Portal branding is best-effort: log the error but still respond with the defaults so the app boots.
		internalctx.GetLogger(ctx).Warn("failed to resolve portal branding", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
	} else if branding != nil {
		resp := mapping.OrganizationBrandingToPortalResponse(*branding)
		response = &resp
	}

	// Branding is resolved from the request Host, so shared caches/CDNs must key on it.
	w.Header().Set("Vary", "Host")
	w.Header().Set("Cache-Control", "public, max-age=60")

	if response != nil {
		RespondJSON(w, response)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// resolvePortalBranding resolves the branding of the organization the host belongs to: self-service
// CustomDomain first, with a fallback to the legacy OrganizationBranding.app_domain column (kept until
// the branding domain migration follow-up ticket). It returns nil when no organization matches the host.
func resolvePortalBranding(ctx context.Context, host string) (*types.OrganizationBranding, error) {
	if branding, err := db.GetOrganizationBrandingByCustomDomain(ctx, host); err != nil {
		return nil, err
	} else if branding != nil {
		return branding, nil
	}
	return db.GetOrganizationBrandingByAppDomain(ctx, host)
}

// normalizeHost lower-cases the host and strips surrounding whitespace, a port and a trailing dot
// so it can be matched against a normalized custom domain / app_domain.
func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(host, ".")
}
