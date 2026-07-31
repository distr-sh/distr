package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

// portalHostSource reports how the request host resolves to an organization. The two custom domain sources are
// kept apart because the instance-scoped OIDC providers stay available on the legacy branding domains: those were
// configured long before self-service domains existed, and their users would otherwise be locked out.
type portalHostSource int

const (
	// portalHostDefault means the host does not belong to any organization, e.g. the instance's default host.
	portalHostDefault portalHostSource = iota
	// portalHostCustomDomain means the host matches a self-service CustomDomain row.
	portalHostCustomDomain
	// portalHostLegacyBranding means the host matches the legacy OrganizationBranding.app_domain column.
	portalHostLegacyBranding
)

func PublicPortalRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Portal"))
	r.Get("/", getPortalHandler).
		With(option.Description("Get the host-resolved portal branding (browser tab title, favicon and logo) " +
			"and the login methods available on this host")).
		With(option.Response(http.StatusOK, api.PortalResponse{}))
}

func getPortalHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	host := validation.NormalizeHostname(r.Host)

	response := api.PortalResponse{}
	branding, source, err := resolvePortalHost(ctx, host)
	if err != nil {
		// Portal branding is best-effort: log the error but still respond with the defaults so the app boots.
		// The host counts as the default host in that case, which keeps the instance login methods available.
		internalctx.GetLogger(ctx).Warn("failed to resolve portal branding", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
	} else {
		if branding != nil {
			response = mapping.OrganizationBrandingToPortalResponse(*branding)
		}
		// Marking the host as a custom domain even without branding is what makes the client drop Distr's own
		// branding when the organization has not configured any of its own.
		response.CustomDomain = source != portalHostDefault
	}
	response.LoginConfig = portalLoginConfig(source)

	// Branding and login methods are resolved from the request Host, so shared caches/CDNs must key on it.
	w.Header().Set("Vary", "Host")
	w.Header().Set("Cache-Control", "public, max-age=60")

	RespondJSON(w, response)
}

// portalLoginConfig lists the login methods offered on a host of the given source. Instance-scoped OIDC providers
// are suppressed on self-service custom domains, where only the organization's own providers apply.
func portalLoginConfig(source portalHostSource) api.PortalLoginConfig {
	resp := api.PortalLoginConfig{RegistrationEnabled: env.Registration() == env.RegistrationEnabled}
	if source == portalHostCustomDomain {
		return resp
	}
	resp.OIDCGithubEnabled = env.OIDCGithubEnabled()
	resp.OIDCGoogleEnabled = env.OIDCGoogleEnabled()
	resp.OIDCMicrosoftEnabled = env.OIDCMicrosoftEnabled()
	resp.OIDCGenericEnabled = env.OIDCGenericEnabled()
	return resp
}

// requestOnCustomDomain reports whether the request host matches a self-service CustomDomain row. Legacy branding
// app domains deliberately do not count: the instance login methods stay available there.
func requestOnCustomDomain(ctx context.Context, r *http.Request) (bool, error) {
	organizationID, err := db.GetOrganizationIDByCustomDomain(ctx, validation.NormalizeHostname(r.Host))
	if err != nil {
		return false, err
	}
	return organizationID != nil, nil
}

// resolvePortalHost resolves the branding of the organization the host belongs to: self-service CustomDomain first,
// with a fallback to the legacy OrganizationBranding.app_domain column (kept until the branding domain migration
// follow-up ticket). The branding may be nil even for a custom domain, which is not the same as the host not
// belonging to an organization: a domain can be registered long before any branding is saved.
func resolvePortalHost(ctx context.Context, host string) (*types.OrganizationBranding, portalHostSource, error) {
	organizationID, err := db.GetOrganizationIDByCustomDomain(ctx, host)
	if err != nil {
		return nil, portalHostDefault, err
	}
	if organizationID != nil {
		branding, err := db.GetOrganizationBranding(ctx, *organizationID)
		if errors.Is(err, apierrors.ErrNotFound) {
			return nil, portalHostCustomDomain, nil
		} else if err != nil {
			return nil, portalHostDefault, err
		}
		return branding, portalHostCustomDomain, nil
	}
	branding, err := db.GetOrganizationBrandingByAppDomain(ctx, host)
	if err != nil {
		return nil, portalHostDefault, err
	}
	if branding == nil {
		return nil, portalHostDefault, nil
	}
	return branding, portalHostLegacyBranding, nil
}
