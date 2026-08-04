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

// portalHost is a request host resolved to an organization. It is the single source of truth for every
// host-dependent decision, so the portal response and the OIDC gating can never disagree about a host.
type portalHost struct {
	source portalHostSource
	// branding is nil unless the host belongs to an organization that has branding configured.
	branding *types.OrganizationBranding
}

// customDomain reports whether the host belongs to an organization through either of the two sources. This is not
// the same as having branding: a domain can be registered long before any branding is saved.
func (h portalHost) customDomain() bool {
	return h.source != portalHostDefault
}

// instanceLoginAllowed reports whether the instance-scoped login methods are offered on this host. They belong to
// the platform rather than to an organization, so a self-service custom domain only gets its organization's own.
func (h portalHost) instanceLoginAllowed() bool {
	return h.source != portalHostCustomDomain
}

func (h portalHost) turnstileSiteKey() *string {
	if h.customDomain() {
		return nil
	}
	return env.TurnstileSiteKey()
}

func PublicPortalRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Portal"))
	r.Get("/", getPortalHandler).
		With(option.Description("Get the host-resolved portal branding (browser tab title, favicon and logo) " +
			"and the login methods available on this host")).
		With(option.Response(http.StatusOK, api.PortalResponse{}))
}

func getPortalHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Resolution is best-effort: on error the host keeps whatever could be determined before it failed, so a
	// failing branding lookup cannot turn a custom domain back into the default host and re-offer instance
	// login methods that do not work there.
	host, err := resolvePortalHost(ctx, validation.NormalizeHostname(r.Host))
	if err != nil {
		internalctx.GetLogger(ctx).Warn("failed to resolve portal host", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
	}

	response := api.PortalResponse{}
	if host.branding != nil {
		response = mapping.OrganizationBrandingToPortalResponse(*host.branding)
	}
	// Marking the host as a custom domain even without branding is what makes the client drop Distr's own
	// branding when the organization has not configured any of its own.
	response.CustomDomain = host.customDomain()
	response.LoginConfig = portalLoginConfig(host)

	// Branding and login methods are resolved from the request Host, so shared caches/CDNs must key on it.
	w.Header().Set("Vary", "Host")
	w.Header().Set("Cache-Control", "public, max-age=60")

	RespondJSON(w, response)
}

// portalLoginConfig lists the login methods offered on the given host. Instance-scoped OIDC providers are
// suppressed on self-service custom domains, where only the organization's own providers apply.
func portalLoginConfig(host portalHost) api.PortalLoginConfig {
	config := api.PortalLoginConfig{
		RegistrationEnabled: env.Registration() == env.RegistrationEnabled,
		TurnstileSiteKey:    host.turnstileSiteKey(),
	}
	if !host.instanceLoginAllowed() {
		return config
	}
	config.OIDCGithubEnabled = env.OIDCGithubEnabled()
	config.OIDCGoogleEnabled = env.OIDCGoogleEnabled()
	config.OIDCMicrosoftEnabled = env.OIDCMicrosoftEnabled()
	config.OIDCGenericEnabled = env.OIDCGenericEnabled()
	return config
}

// resolvePortalHost resolves the (normalized) host to the organization it belongs to: self-service CustomDomain
// first, with a fallback to the legacy OrganizationBranding.app_domain column (kept until the branding domain
// migration follow-up ticket). On error it still returns everything that was resolved before the failure, so
// callers that treat resolution as best-effort do not silently downgrade a custom domain to the default host.
func resolvePortalHost(ctx context.Context, host string) (portalHost, error) {
	organizationID, err := db.GetOrganizationIDByCustomDomain(ctx, host)
	if err != nil {
		return portalHost{}, err
	}
	if organizationID != nil {
		resolved := portalHost{source: portalHostCustomDomain}
		branding, err := db.GetOrganizationBranding(ctx, *organizationID)
		if errors.Is(err, apierrors.ErrNotFound) {
			return resolved, nil
		} else if err != nil {
			return resolved, err
		}
		resolved.branding = branding
		return resolved, nil
	}
	branding, err := db.GetOrganizationBrandingByAppDomain(ctx, host)
	if err != nil {
		return portalHost{}, err
	}
	if branding == nil {
		return portalHost{}, nil
	}
	return portalHost{source: portalHostLegacyBranding, branding: branding}, nil
}
