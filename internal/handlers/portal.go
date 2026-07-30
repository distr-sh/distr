package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
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
			option.ContentDescription("The host did not resolve to a custom app domain, or it could not be resolved. "+
				"Clients are instructed to apply default branding. A custom app domain whose organization has not "+
				"configured any branding is answered with an empty 200 response instead.")))
}

func getPortalHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	host := validation.NormalizeHostname(r.Host)

	var response *api.PortalResponse
	if branding, customDomain, err := resolvePortalBranding(ctx, host); err != nil {
		// Portal branding is best-effort: log the error but still respond with the defaults so the app boots.
		internalctx.GetLogger(ctx).Warn("failed to resolve portal branding", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
	} else if customDomain {
		// An empty response still marks the host as a custom domain, which is what makes the client
		// drop Distr's own branding even when the organization has not configured any of its own.
		resp := api.PortalResponse{}
		if branding != nil {
			resp = mapping.OrganizationBrandingToPortalResponse(*branding)
		}
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
// the branding domain migration follow-up ticket). The second return value reports whether the host
// belongs to an organization at all, which is not the same as having branding: a custom domain can be
// registered long before any branding is saved.
func resolvePortalBranding(ctx context.Context, host string) (*types.OrganizationBranding, bool, error) {
	organizationID, err := db.GetOrganizationIDByCustomDomain(ctx, host)
	if err != nil {
		return nil, false, err
	}
	if organizationID != nil {
		branding, err := db.GetOrganizationBranding(ctx, *organizationID)
		if errors.Is(err, apierrors.ErrNotFound) {
			return nil, true, nil
		} else if err != nil {
			return nil, false, err
		}
		return branding, true, nil
	}
	branding, err := db.GetOrganizationBrandingByAppDomain(ctx, host)
	if err != nil {
		return nil, false, err
	}
	return branding, branding != nil, nil
}
